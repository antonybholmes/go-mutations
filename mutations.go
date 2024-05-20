package mutations

import (
	"database/sql"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/antonybholmes/go-dna"
	"github.com/antonybholmes/go-sys"
	"github.com/rs/zerolog/log"
)

const META_SQL = `SELECT samples FROM metadata`

const FIND_MUTATIONS_SQL = `SELECT 
	chr, 
	start, 
	end, 
	ref, 
	tum, 
	t_alt_count, 
	t_depth, 
	variant_type,
	vaf,
	sample, 
	dataset
	FROM maf 
	WHERE chr = ?1 AND start >= ?2 AND end <= ?3 
	ORDER BY chr, start, end, variant_type, dataset`

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
	Id          string `json:"id"`
	Name        string `json:"name"`
	Assembly    string `json:"assembly"`
	Description string `json:"description"`
	Samples     uint   `json:"samples"`
}

type Mutation struct {
	Chr         string  `json:"chr"`
	Start       uint    `json:"start"`
	End         uint    `json:"end"`
	Ref         string  `json:"ref"`
	Tum         string  `json:"tum"`
	Alt         int     `json:"alt,omitempty"`
	Depth       int     `json:"depth,omitempty"`
	VariantType string  `json:"type,omitempty"`
	Vaf         float32 `json:"vaf,omitempty"`
	Sample      string  `json:"sample"`
	Dataset     string  `json:"dataset,omitempty"`
}

type MutationResults struct {
	Location  *dna.Location `json:"location"`
	DB        *MutationSet  `json:"db"`
	Mutations []*Mutation   `json:"mutations"`
}

type Pileup struct {
	Location    *dna.Location `json:"location"`
	MutationSet *MutationSet  `json:"mutationSet"`
	Mutations   [][]*Mutation `json:"mutations"`
}

type MutationDBCache struct {
	dir      string
	cacheMap map[string]map[string]*MutationDB
}

func NewMutationSet(assembly string, name string) *MutationSet {
	id := fmt.Sprintf("%s:%s", assembly, name)
	return &MutationSet{Id: id, Assembly: assembly, Name: name, Description: "", Samples: 0}
}

func NewMutationDBCache(dir string) *MutationDBCache {

	cacheMap := make(map[string]map[string]*MutationDB)

	assemblyFiles, err := os.ReadDir(dir)

	if err != nil {
		log.Fatal().Msgf("error opening %s", dir)
	}

	for _, assemblyFile := range assemblyFiles {
		if assemblyFile.IsDir() {
			dbFiles, err := os.ReadDir(filepath.Join(dir, assemblyFile.Name()))

			if err != nil {
				log.Fatal().Msgf("%s", err)

			}

			// init the cache
			cacheMap[assemblyFile.Name()] = make(map[string]*MutationDB)

			for _, dbFile := range dbFiles {

				mutationSet := NewMutationSet(assemblyFile.Name(), dbFile.Name())

				path := filepath.Join(dir, mutationSet.Assembly, mutationSet.Name)

				db, err := NewMutationDB(path, mutationSet)

				if err != nil {
					log.Fatal().Msgf("%s", err)
				}

				//key := fmt.Sprintf("%s:%s", mutationSet.Assembly, mutationSet.Name)

				cacheMap[mutationSet.Assembly][mutationSet.Name] = db

			}
		}

	}

	return &MutationDBCache{dir, cacheMap}
}

func (cache *MutationDBCache) Dir() string {
	return cache.dir
}

func (cache *MutationDBCache) List() []*MutationSet {

	ret := make([]*MutationSet, 0, len(cache.cacheMap))

	assemblies := make([]string, 0, len(cache.cacheMap))

	for assembly := range cache.cacheMap {
		assemblies = append(assemblies, assembly)
	}
	sort.Strings(assemblies)

	log.Debug().Msgf("%s blob", assemblies)

	for _, assembly := range assemblies {
		dbs := make([]string, 0, len(cache.cacheMap[assembly]))

		for db := range cache.cacheMap[assembly] {
			dbs = append(dbs, db)
		}
		sort.Strings(dbs)

		log.Debug().Msgf("%s blob2", dbs)

		for _, db := range dbs {
			ret = append(ret, cache.cacheMap[assembly][db].MutationSet())
		}
	}

	return ret
}

func (cache *MutationDBCache) MutationDBFromId(id string) (*MutationDB, error) {
	tokens := strings.Split(id, ":")

	if len(tokens) < 2 {
		return nil, fmt.Errorf("mutations %s is not a valid id", id)
	}

	assembly := tokens[0]
	name := tokens[1]

	return cache.MutationDB(assembly, name)
}

func (cache *MutationDBCache) MutationDBFromMutationSet(mutationSet *MutationSet) (*MutationDB, error) {
	return cache.MutationDB(mutationSet.Assembly, mutationSet.Name)
}

func (cache *MutationDBCache) MutationDB(assembly string, name string) (*MutationDB, error) {

	_, ok := cache.cacheMap[assembly]

	if !ok {
		return nil, fmt.Errorf("mutations: %s is not a valid assembly", assembly)
	}

	_, ok = cache.cacheMap[assembly][name]

	if !ok {
		return nil, fmt.Errorf("mutations: %s is not a valid name", name)
	}

	return cache.cacheMap[assembly][name], nil
}

func (cache *MutationDBCache) Close() {
	for _, dbs := range cache.cacheMap {
		for _, db := range dbs {
			db.Close()
		}
	}
}

type MutationDB struct {
	mutationSet       *MutationSet
	Path              string
	db                *sql.DB
	findMutationsStmt *sql.Stmt
}

func NewMutationDB(dir string, mutationSet *MutationSet) (*MutationDB, error) {
	db := sys.Must(sql.Open("sqlite3", path.Join(dir, "maf.db")))

	metaStmt := sys.Must(db.Prepare(META_SQL))

	err := metaStmt.QueryRow().Scan(
		&mutationSet.Samples)

	if err != nil {
		log.Fatal().Msgf("%s", err)
	}

	return &MutationDB{
		mutationSet:       mutationSet,
		db:                db,
		Path:              dir,
		findMutationsStmt: sys.Must(db.Prepare(FIND_MUTATIONS_SQL)),
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

func (db *MutationDB) MutationSet() *MutationSet {
	return db.mutationSet
}

func (db *MutationDB) FindMutations(location *dna.Location) (*MutationResults, error) {

	rows, err := db.findMutationsStmt.Query(location.Chr, location.Start, location.End)

	if err != nil {
		return nil, err
	}

	return rowsToMutations(location, db.mutationSet, rows)
}

func (db *MutationDB) Pileup(location *dna.Location) (*Pileup, error) {

	rows, err := db.findMutationsStmt.Query(location.Chr, location.Start, location.End)

	if err != nil {
		return nil, err
	}

	results, err := rowsToMutations(location, db.mutationSet, rows)

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

	return &Pileup{location, db.mutationSet, pileup}, nil
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
	mutationsdb.db.Close()
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
			&mutation.Tum,
			&mutation.Alt,
			&mutation.Depth,
			&mutation.VariantType,
			&mutation.Vaf,
			&mutation.Sample,
			&mutation.Dataset)

		if err != nil {
			fmt.Println(err)
		}

		mutations = append(mutations, &mutation)
	}

	return &MutationResults{location, mutationSet, mutations}, nil
}
