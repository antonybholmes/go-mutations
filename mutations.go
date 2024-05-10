package mutations

import (
	"database/sql"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/antonybholmes/go-dna"
	"github.com/antonybholmes/go-sys"
)

const ALL_PLATFORMS_SQL = `SELECT uuid, name
	FROM platforms ORDER BY name`

const PLATFORM_SAMPLES_SQL = `SELECT uuid, name
	FROM samples WHERE platform = ?1 ORDER BY name`

const FIND_MUTATIONS_SQL = `SELECT chr, start, end, ref, tum, variant_type, sample
	FROM maf WHERE chr = ?1 AND start >= ?2 AND end <= ?3 ORDER BY chr, start, end, variant_type`

type ExpressionDataIndex struct {
	ProbeIds    []string
	EntrezIds   []string
	GeneSymbols []string
}

type ExpressionData struct {
	Exp    [][]float64
	Header []string
	Index  ExpressionDataIndex
}

type MutationsReq struct {
	Assembly string        `json:"assembly"`
	Location *dna.Location `json:"location"`
	Samples  []string      `json:"samples"`
}

type MutationSet struct {
	Uuid     string `json:"uuid"`
	Name     string `json:"name"`
	Assembly string `json:"assembly"`
}

type Mutation struct {
	Chr         string  `json:"chr"`
	Start       uint    `json:"start"`
	End         uint    `json:"end"`
	Ref         string  `json:"ref"`
	Mut         string  `json:"mut"`
	VariantType string  `json:"variantType"`
	Vaf         float32 `json:"vaf"`
	Sample      string  `json:"sample"`
}

type MutationResults struct {
	Location  *dna.Location `json:"locaation"`
	DB        *MutationSet  `json:"db"`
	Mutations []*Mutation   `json:"mutations"`
}

type Pileup struct {
	Location  *dna.Location `json:"locaation"`
	DB        *MutationSet  `json:"db"`
	Mutations [][]*Mutation `json:"mutations"`
}

type MutationDBCache struct {
	dir      string
	cacheMap *map[string]*MutationDB
}

func NewMutationDBCache(dir string) *MutationDBCache {
	assemblyEntries, err := os.ReadDir(dir)

	if err != nil {
		log.Fatal().Msgf("%s", err)
	}

	cacheMap := make(map[string]*MutationDB)

	for _, assemblyEntry := range assemblyEntries {
		if assemblyEntry.IsDir() {
			setEntries, err := os.ReadDir(assemblyEntry.Name())

			if err != nil {
				log.Fatal().Msgf("%s", err)
			}

			for _, setEntry := range setEntries {
				mutationSet := MutationSet{Name: setEntry.Name(), Assembly: assemblyEntry.Name()}

				db, err := NewMutationDB(filepath.Join(dir, mutationSet.Assembly, fmt.Sprintf("%s.db", mutationSet.Name)), &mutationSet)

				if err != nil {
					log.Fatal().Msgf("%s", err)
				}

				key := fmt.Sprintf("%s:%s", mutationSet.Assembly, mutationSet.Name)

				cacheMap[key] = db

			}
		}

	}

	return &MutationDBCache{dir: dir,
		cacheMap: &cacheMap}
}

func (cache *MutationDBCache) Dir() string {
	return cache.dir
}

func (cache *MutationDBCache) MutationDB(mutationSet *MutationSet) (*MutationDB, error) {
	key := fmt.Sprintf("%s:%s", mutationSet.Assembly, mutationSet.Name)

	_, ok := (*cache.cacheMap)[key]

	if !ok {
		db, err := NewMutationDB(filepath.Join(cache.dir, fmt.Sprintf("%s.db", mutationSet.Assembly)), mutationSet)

		if err != nil {
			return nil, err
		}

		(*cache.cacheMap)[key] = db
	}

	return (*cache.cacheMap)[key], nil
}

func (cache *MutationDBCache) Close() {
	for _, db := range *cache.cacheMap {
		db.Close()
	}
}

type MutationDB struct {
	MutationSet         *MutationSet
	Path                string
	DB                  *sql.DB
	AllMutationSetsStmt *sql.Stmt
	AllSamplesStmt      *sql.Stmt
	FindMutationsStmt   *sql.Stmt
}

func NewMutationDB(dir string, mutationSet *MutationSet) (*MutationDB, error) {
	db := sys.Must(sql.Open("sqlite3", path.Join(dir, "mutations.db")))

	return &MutationDB{
		MutationSet:         mutationSet,
		DB:                  db,
		Path:                dir,
		AllMutationSetsStmt: sys.Must(db.Prepare(ALL_PLATFORMS_SQL)),
		AllSamplesStmt:      sys.Must(db.Prepare(PLATFORM_SAMPLES_SQL)),
		FindMutationsStmt:   sys.Must(db.Prepare(FIND_MUTATIONS_SQL)),
	}, nil
}

// func (mutationsdb *MutationsDB) AllMutationSets() (*[]MutationSet, error) {

// 	rows, err := mutationsdb.AllMutationSetsStmt.Query()

// 	if err != nil {
// 		return nil, err
// 	}

// 	mutationSets := []MutationSet{}

// 	defer rows.Close()

// 	for rows.Next() {
// 		var mutationSet MutationSet
// 		err := rows.Scan(&mutationSet.Uuid, &mutationSet.Name)

// 		if err != nil {
// 			fmt.Println(err)
// 		}

// 		mutationSets = append(mutationSets, mutationSet)
// 	}

// 	return &mutationSets, nil
// }

func (db *MutationDB) FindMutations(location *dna.Location) (*MutationResults, error) {

	rows, err := db.FindMutationsStmt.Query(location.Chr, location.Start, location.End)

	if err != nil {
		return nil, err
	}

	return rowsToMutations(location, db.MutationSet, rows)
}

func (db *MutationDB) Pileup(location *dna.Location) (*Pileup, error) {

	rows, err := db.FindMutationsStmt.Query(location.Chr, location.Start, location.End)

	if err != nil {
		return nil, err
	}

	results, err := rowsToMutations(location, db.MutationSet, rows)

	if err != nil {
		return nil, err
	}

	pileup := make([][]*Mutation, location.Len())

	for i := range location.Len() {
		pileup[i] = []*Mutation{}
	}

	for _, mutation := range results.Mutations {
		offset := mutation.Start - location.Start

		pileup[offset] = append(pileup[offset], mutation)
	}

	return &Pileup{location, db.MutationSet, pileup}, nil
}

// func (mutationsdb *MutationsDB) Expression(samples *MutationsReq) (*ExpressionData, error) {
// 	// let sample_ids = vec![
// 	//     "0c3b8a19-1975-4c6e-aece-44a59c71719d",
// 	//     "0c4f0c89-af16-484a-a408-8dfde25d8f10",
// 	// ];

// 	nSamples := len(samples.Samples)

// 	//eprintln!("{:?}", Path::new(&self.path).join("meta.tsv").to_str());
// 	// open meta data
// 	file, err := os.Open(path.Join(mutationsdb.Path, samples.Platform, "meta.tsv"))

// 	if err != nil {
// 		return nil, err
// 	}

// 	reader := csv.NewReader(file)
// 	reader.Comma = '\t'

// 	columnValues, err := reader.Read()

// 	if err != nil {
// 		return nil, err
// 	}

// 	probeIds := columnValues[1:]

// 	nProbes := len(probeIds)

// 	columnValues, err = reader.Read()

// 	if err != nil {
// 		return nil, err
// 	}

// 	entrezIds := columnValues[1:]

// 	columnValues, err = reader.Read()

// 	if err != nil {
// 		return nil, err
// 	}

// 	geneSymbols := columnValues[1:]

// 	rowRecords := make([][]float64, nSamples)
// 	sampleNames := make([]string, nSamples)

// 	// let mut rdr = csv::ReaderBuilder::new()
// 	//     .has_headers(true)
// 	//     .delimiter(b'\t')
// 	//     .from_reader(file);

// 	for _, sample := range samples.Samples {
// 		file, err := os.Open(path.Join(mutationsdb.Path, fmt.Sprintf("%s.tsv", sample)))

// 		if err != nil {
// 			return nil, err
// 		}

// 		reader := csv.NewReader(file)
// 		reader.Comma = '\t'

// 		columnValues, err := reader.Read()

// 		if err != nil {
// 			return nil, err
// 		}

// 		sampleNames = append(sampleNames, columnValues[0])

// 		for {
// 			columnValues, err := reader.Read()

// 			if err == io.EOF {
// 				break
// 			}

// 			if err != nil {
// 				return nil, err
// 			}

// 			rowRecords = append(rowRecords, sys.Map(columnValues[1:], func(s string) float64 {
// 				v, err := strconv.ParseFloat(s, 64)

// 				if err != nil {
// 					return 0
// 				}

// 				return v
// 			}))
// 		}
// 	}

// 	data := ExpressionData{
// 		Exp:    make([][]float64, nProbes),
// 		Header: sampleNames,
// 		Index: ExpressionDataIndex{
// 			ProbeIds:    probeIds,
// 			EntrezIds:   entrezIds,
// 			GeneSymbols: geneSymbols,
// 		},
// 	}

// 	// We are going to transpose the data we read into this output
// 	// array
// 	//let mut data:Vec<Vec<f64>> = vec![vec![0.0; n_samples]; n_probes] ;

// 	for row := range nProbes {
// 		data.Exp[row] = make([]float64, nSamples)

// 		for col := range nSamples {
// 			data.Exp[row][col] = rowRecords[col][row]
// 		}

// 		//eprintln!("{:?} {} out_row", out_row, n_samples);

// 		// wtr.write_record(&out_row)?;
// 	}

// 	// for result in rdr.records() {
// 	//     //for row in data {
// 	//     let record =
// 	//         result.map_err(|_| MicroarrayError::FileError("header issue".to_string()))?;

// 	//     let row = cols.iter().map(|c| &record[**c]).collect::<Vec<&str>>();
// 	//     //println!("{}", &record[0]);
// 	//     wtr.write_record(&row)
// 	//         .map_err(|_| MicroarrayError::FileError("header issue".to_string()))?;
// 	// }

// 	//let vec: Vec<u8> = wtr.into_inner()?;

// 	//let data = String::from_utf8(vec)?;

// 	return &data, nil
// }

func (mutationsdb *MutationDB) Close() {
	mutationsdb.DB.Close()
}

func rowsToMutations(location *dna.Location, mutationSet *MutationSet, rows *sql.Rows) (*MutationResults, error) {

	mutations := []*Mutation{}

	defer rows.Close()

	for rows.Next() {
		var mutation Mutation

		err := rows.Scan(
			&mutation.Chr,
			&mutation.Start,
			&mutation.End,
			&mutation.Ref,
			&mutation.Mut,
			&mutation.VariantType,
			&mutation.Vaf,
			&mutation.Sample)

		if err != nil {
			fmt.Println(err)
		}

		mutations = append(mutations, &mutation)
	}

	return &MutationResults{location, mutationSet, mutations}, nil
}
