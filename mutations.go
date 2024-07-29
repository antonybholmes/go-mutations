package mutations

import (
	"database/sql"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/antonybholmes/go-dna"
	"github.com/rs/zerolog/log"
)

const META_SQL = "SELECT public_id, name, description, assembly FROM metadata"

const DATASETS_SQL = `SELECT
	name
	FROM datasets 
	ORDER BY datasets.name`

const SAMPLES_SQL = `SELECT 
	name, 
	coo, 
	lymphgen, 
	paired_normal_dna, 
	institution, 
	sample_type,
	dataset
	FROM samples 
	ORDER BY samples.name`

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
	FROM mutations 
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

type MutationDBMetadata struct {
	Id          string `json:"id"`
	PublicId    string `json:"publicId"`
	Name        string `json:"name"`
	Assembly    string `json:"assembly"`
	Description string `json:"description"`
	Samples     uint   `json:"samples"`
}

// type MutationDBDataSet struct {
// 	Name string `json:"name"`
// }

type MutationDBSample struct {
	Uuid            string `json:"uuid"`
	Name            string `json:"name"`
	COO             string `json:"coo"`
	Lymphgen        string `json:"lymphgen"`
	PairedNormalDna bool   `json:"pairedNormalDna"`
	Institution     string `json:"institution"`
	SampleType      string `json:"sampleType"`
	Dataset         string `json:"dataset"`
}

type MutationDBInfo struct {
	Metadata *MutationDBMetadata `json:"metadata"`
	//Datasets []*MutationDBDataSet `json:"datasets"`
	Samples []*MutationDBSample `json:"samples"`
}

type Mutation struct {
	Chr     string  `json:"chr"`
	Start   uint    `json:"start"`
	End     uint    `json:"end"`
	Ref     string  `json:"ref"`
	Tum     string  `json:"tum"`
	Alt     int     `json:"alt,omitempty"`
	Depth   int     `json:"depth,omitempty"`
	Type    string  `json:"type,omitempty"`
	Vaf     float32 `json:"vaf,omitempty"`
	Sample  string  `json:"sample"`
	Dataset string  `json:"dataset"`
}

type MutationResults struct {
	Location *dna.Location `json:"location"`

	Mutations []*Mutation `json:"mutations"`
}

type Pileup struct {
	Location *dna.Location `json:"location"`

	//Samples   uint                  `json:"samples"`
	Mutations [][]*Mutation `json:"mutations"`
}

type MutationDBCache struct {
	dir      string
	cacheMap map[string]*MutationDB
}

func MutationDBKey(assembly string, name string) string {
	return fmt.Sprintf("%s:%s", assembly, name)
}

func NewMutationDBMetaData(assembly string, name string) *MutationDBMetadata {

	return &MutationDBMetadata{
		Id:          MutationDBKey(assembly, name),
		Assembly:    assembly,
		Name:        name,
		Description: "",
	}
}

func NewMutationDBCache(dir string) *MutationDBCache {

	cacheMap := make(map[string]*MutationDB)

	assemblyFiles, err := os.ReadDir(dir)

	log.Debug().Msgf("---- mutations ----")

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
			//cacheMap[assemblyFile.Name()] = make(map[string]*MutationDB)

			for _, dbFile := range dbFiles {

				log.Debug().Msgf("Loading mutations from %s...", dbFile.Name())

				//metadata := NewMutationDBMetaData(assemblyFile.Name(), dbFile.Name())

				path := filepath.Join(dir, assemblyFile.Name(), dbFile.Name())

				db, err := NewMutationDB(path)

				if err != nil {
					log.Fatal().Msgf("%s", err)
				}

				log.Debug().Msgf("Caching %s", db.Info.Metadata.Id)

				cacheMap[db.Info.Metadata.Id] = db
			}
		}
	}

	log.Debug().Msgf("---- end ----")

	return &MutationDBCache{dir, cacheMap}
}

func (cache *MutationDBCache) Dir() string {
	return cache.dir
}

func (cache *MutationDBCache) List() []*MutationDBInfo {

	ret := make([]*MutationDBInfo, 0, len(cache.cacheMap))

	ids := make([]string, 0, len(cache.cacheMap))

	for id := range cache.cacheMap {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	for _, id := range ids {
		ret = append(ret, cache.cacheMap[id].Info)
	}

	return ret
}

func (cache *MutationDBCache) MutationDBFromId(id string) (*MutationDB, error) {
	db, ok := cache.cacheMap[id]

	if !ok {
		return nil, fmt.Errorf("mutations: %s is not a valid assembly", id)
	}

	return db, nil
}

func (cache *MutationDBCache) MutationDBFromMetadata(metadata *MutationDBMetadata) (*MutationDB, error) {
	return cache.MutationDB(metadata.Assembly, metadata.Name)
}

func (cache *MutationDBCache) MutationDB(assembly string, name string) (*MutationDB, error) {
	return cache.MutationDBFromId(MutationDBKey(assembly, name))
}

func (cache *MutationDBCache) Close() {
	// for _, db := range cache.cacheMap {
	// 	db.Close()
	// }
}

type MutationDB struct {
	Info *MutationDBInfo `json:"info"`

	File string
	//db                *sql.DB
	//findMutationsStmt *sql.Stmt
}

func NewMutationDB(dir string) (*MutationDB, error) {
	file := path.Join(dir, "mutations.db")
	db, err := sql.Open("sqlite3", file)

	if err != nil {
		log.Fatal().Msgf("%s", err)
	}

	defer db.Close()

	metadata := &MutationDBMetadata{}

	err = db.QueryRow(META_SQL).Scan(&metadata.PublicId, &metadata.Name, &metadata.Description, &metadata.Assembly)

	if err != nil {
		log.Fatal().Msgf("%s", err)
	}

	metadata.Id = MutationDBKey(metadata.Assembly, metadata.PublicId)

	// datasetRows, err := db.Query(DATASETS_SQL)

	// if err != nil {
	// 	log.Fatal().Msgf("%s", err)
	// }

	// defer datasetRows.Close()

	// datasets := []*MutationDBDataSet{}

	// for datasetRows.Next() {
	// 	var dataset MutationDBDataSet

	// 	err := datasetRows.Scan(
	// 		&dataset.Name)

	// 	if err != nil {
	// 		log.Fatal().Msgf("%s", err)
	// 	}

	// 	datasets = append(datasets, &dataset)
	// }

	sampleRows, err := db.Query(SAMPLES_SQL)

	if err != nil {
		log.Fatal().Msgf("%s", err)
	}

	defer sampleRows.Close()

	samples := []*MutationDBSample{}

	for sampleRows.Next() {
		var sample MutationDBSample

		err := sampleRows.Scan(
			&sample.Name,
			&sample.COO,
			&sample.Lymphgen,
			&sample.PairedNormalDna,
			&sample.Institution,
			&sample.SampleType,
			&sample.Dataset)

		if err != nil {
			log.Fatal().Msgf("%s", err)
		}

		samples = append(samples, &sample)
	}

	info := &MutationDBInfo{
		Metadata: metadata,
		//Datasets: datasets,
		Samples: samples,
	}

	return &MutationDB{
		Info: info,
		//db:                db,
		File: file,
		//findMutationsStmt: sys.Must(db.Prepare(FIND_MUTATIONS_SQL)),
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

func (mutdb *MutationDB) FindMutations(location *dna.Location) (*MutationResults, error) {

	db, err := sql.Open("sqlite3", mutdb.File) //not clear on what is needed for the user and password

	if err != nil {
		return nil, err
	}

	defer db.Close()

	rows, err := db.Query(FIND_MUTATIONS_SQL, location.Chr, location.Start, location.End)

	if err != nil {
		return nil, err
	}

	return rowsToMutations(location, rows)
}

func (mutdb *MutationDB) Pileup(location *dna.Location) (*Pileup, error) {

	db, err := sql.Open("sqlite3", mutdb.File) //not clear on what is needed for the user and password

	if err != nil {
		return nil, err
	}

	defer db.Close()

	rows, err := db.Query(FIND_MUTATIONS_SQL, location.Chr, location.Start, location.End)

	if err != nil {
		return nil, err
	}

	results, err := rowsToMutations(location, rows)

	if err != nil {
		return nil, err
	}

	// first lets fix deletions and insertions

	for _, mutation := range results.Mutations {
		// change for sorting purposes so that ins always comes last
		switch mutation.Type {
		case "INS":
			mutation.Type = "2:INS"
			// modify the output so that is begins with a caret to indicate
			// an insertion
			mutation.Tum = fmt.Sprintf("^%s", mutation.Tum)
		case "DEL":
			mutation.Type = "3:DEL"
		default:
			mutation.Type = "1:SNP"
		}
	}

	// put together by position, type, tum

	pileupMap := make(map[uint]map[string]map[string][]*Mutation)

	for _, mutation := range results.Mutations {

		switch mutation.Type {
		case "3:DEL":
			for i := range mutation.End - mutation.Start + 1 {
				addToPileupMap(&pileupMap, mutation.Start+i, mutation)
			}
		case "2:INS":
			addToPileupMap(&pileupMap, mutation.Start, mutation)
		default:
			// deal with concatenated snps
			tum := []rune(mutation.Tum)
			for i := range len(tum) {
				// clone and change tumor
				mut := *mutation
				mut.Tum = string(tum[i])
				addToPileupMap(&pileupMap, mutation.Start+uint(i), &mut)
			}
		}
	}

	// init pileup
	pileup := make([][]*Mutation, location.Len())

	for i := range location.Len() {
		pileup[i] = []*Mutation{}
	}

	starts := make([]uint, 0, len(pileupMap))

	for start := range pileupMap {
		starts = append(starts, start)
	}

	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })

	for _, start := range starts {
		variantTypes := make([]string, 0, len(pileupMap[start]))

		for variantType := range pileupMap[start] {
			variantTypes = append(variantTypes, variantType)
		}

		sort.Strings(variantTypes)

		for _, variantType := range variantTypes {
			tumors := make([]string, 0, len(pileupMap[start][variantType]))

			for tumor := range pileupMap[start][variantType] {
				tumors = append(tumors, tumor)
			}

			sort.Strings(tumors)

			for _, tumor := range tumors {
				mutations := pileupMap[start][variantType][tumor]

				for _, mutation := range mutations {
					offset := start - location.Start

					pileup[offset] = append(pileup[offset], mutation)
				}

			}

		}
	}

	return &Pileup{location, pileup}, nil
}

func addToPileupMap(pileupMap *map[uint]map[string]map[string][]*Mutation, start uint, mutation *Mutation) {

	_, ok := (*pileupMap)[start]

	if !ok {
		(*pileupMap)[start] = make(map[string]map[string][]*Mutation)
	}

	_, ok = (*pileupMap)[start][mutation.Type]

	if !ok {
		(*pileupMap)[start][mutation.Type] = make(map[string][]*Mutation)
	}

	_, ok = (*pileupMap)[start][mutation.Type][mutation.Tum]

	if !ok {
		(*pileupMap)[start][mutation.Type][mutation.Tum] = make([]*Mutation, 0, 10)
	}

	(*pileupMap)[start][mutation.Type][mutation.Tum] = append((*pileupMap)[start][mutation.Type][mutation.Tum], mutation)
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

// func (mutationsdb *MutationDB) Close() {
// 	mutationsdb.db.Close()
// }

func rowsToMutations(location *dna.Location, rows *sql.Rows) (*MutationResults, error) {

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
			&mutation.Type,
			&mutation.Vaf,
			&mutation.Sample,
			&mutation.Dataset)

		if err != nil {
			fmt.Println(err)
		}

		mutations = append(mutations, &mutation)
	}

	return &MutationResults{location, mutations}, nil
}
