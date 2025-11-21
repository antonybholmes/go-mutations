package mutations

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"slices"
	"sort"
	"strings"

	"github.com/antonybholmes/go-dna"
	"github.com/rs/zerolog/log"
)

type (
	ExpressionDataIndex struct {
		ProbeIds    []string
		EntrezIds   []string
		GeneSymbols []string
	}

	ExpressionData struct {
		Exp    [][]float64
		Header []string
		Index  ExpressionDataIndex
	}

	MutationsReq struct {
		Assembly string        `json:"assembly"`
		Location *dna.Location `json:"location"`
		Samples  []string      `json:"samples"`
	}

	Dataset struct {
		File        string    `json:"-"`
		PublicId    string    `json:"publicId"`
		ShortName   string    `json:"shortName"`
		Name        string    `json:"name"`
		Assembly    string    `json:"assembly"`
		Description string    `json:"description"`
		Samples     []*Sample `json:"samples"`

		//db                *sql.DB
		//findMutationsStmt *sql.Stmt
	}

	Sample struct {
		PublicId        string `json:"publicId"`
		Name            string `json:"name"`
		COO             string `json:"coo"`
		Lymphgen        string `json:"lymphgen"`
		Institution     string `json:"institution"`
		SampleType      string `json:"sampleType"`
		Dataset         string `json:"dataset"`
		Id              int    `json:"id"`
		PairedNormalDna int    `json:"pairedNormalDna"`
	}

	Mutation struct {
		Chr     string  `json:"chr"`
		Ref     string  `json:"ref"`
		Tum     string  `json:"tum"`
		Type    string  `json:"type"`
		Sample  string  `json:"sample"`
		Dataset string  `json:"dataset,omitempty"`
		Start   int     `json:"start"`
		End     int     `json:"end"`
		Alt     int     `json:"tAltCount"`
		Depth   int     `json:"tDepth"`
		Vaf     float32 `json:"vaf"`
	}

	DatasetResults struct {
		Dataset string `json:"dataset"`

		Mutations []*Mutation `json:"mutations"`
	}

	PileupResults struct {
		Location *dna.Location `json:"location"`
		Datasets []string      `json:"datasets"`
		//Samples   int                  `json:"samples"`
		Pileup [][]*Mutation `json:"pileup"`
	}

	SearchResults struct {
		Location *dna.Location `json:"location"`
		//Info           []*Info           `json:"info"`
		DatasetResults []*DatasetResults `json:"results"`
	}

	DatasetCache struct {
		cacheMap map[string]map[string]*Dataset
		dir      string
	}
)

const (
	InfoSql = "SELECT public_id, short_name, name, description, assembly FROM info"

	// const DATASETS_SQL = `SELECT
	// 	name
	// 	FROM datasets
	// 	ORDER BY datasets.name`

	SampleSql = `SELECT
		id,
		public_id,
		name, 
		coo, 
		lymphgen, 
		paired_normal_dna, 
		institution, 
		sample_type
		FROM samples 
		ORDER BY samples.name`

	FindMutationsSql = `SELECT 
		chr, 
		start, 
		end, 
		ref, 
		tum, 
		t_alt_count, 
		t_depth, 
		variant_type,
		vaf,
		sample_public_id
		FROM mutations 
		WHERE chr = :chr AND start >= :start AND end <= :end 
		ORDER BY chr, start, end, variant_type`
)

func (mutation *Mutation) Clone() *Mutation {
	var ret Mutation = Mutation{Chr: mutation.Chr,
		Start:  mutation.Start,
		End:    mutation.End,
		Ref:    mutation.Ref,
		Tum:    mutation.Tum,
		Alt:    mutation.Alt,
		Depth:  mutation.Depth,
		Type:   mutation.Type,
		Vaf:    mutation.Vaf,
		Sample: mutation.Sample,
	}

	return &ret
}

// func MutationDBKey(assembly string, name string) string {
// 	return fmt.Sprintf("%s:%s", assembly, name)
// }

// func NewMutationDBMetaData(assembly string, name string) *MutationDBMetadata {

// 	return &MutationDBMetadata{
// 		Id:          MutationDBKey(assembly, name),
// 		Assembly:    assembly,
// 		Name:        name,
// 		Description: "",
// 	}
// }

// type Dataset struct {
// 	Info    *Info     `json:"info"`
// 	Samples []*Sample `json:"samples"`
// }

func NewDataset(file string) (*Dataset, error) {
	//file := path.Join(dir, "mutations.db")
	db, err := sql.Open("sqlite3", file)

	if err != nil {
		log.Fatal().Msgf("%s", err)
	}

	defer db.Close()

	dataset := &Dataset{
		File:    file,
		Samples: make([]*Sample, 0, 100),
	}

	err = db.QueryRow(InfoSql).Scan(&dataset.PublicId,
		&dataset.ShortName,
		&dataset.Name,
		&dataset.Description,
		&dataset.Assembly)

	if err != nil {
		log.Fatal().Msgf("info %s", err)
	}

	//mutationDB.Id = MutationDBKey(mutationDB.Assembly, mutationDB.PublicId)

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

	sampleRows, err := db.Query(SampleSql)

	if err != nil {
		log.Fatal().Msgf("%s", err)
	}

	defer sampleRows.Close()

	for sampleRows.Next() {
		var sample Sample

		err := sampleRows.Scan(
			&sample.Id,
			&sample.PublicId,
			&sample.Name,
			&sample.COO,
			&sample.Lymphgen,
			&sample.PairedNormalDna,
			&sample.Institution,
			&sample.SampleType)

		sample.Dataset = dataset.PublicId

		if err != nil {
			log.Fatal().Msgf("%s", err)
		}

		dataset.Samples = append(dataset.Samples, &sample)
	}

	return dataset, nil
}

func (dataset *Dataset) Search(location *dna.Location) (*DatasetResults, error) {

	db, err := sql.Open("sqlite3", dataset.File) //not clear on what is needed for the user and password

	if err != nil {
		return nil, err
	}

	defer db.Close()

	// need to search without chr prefix

	rows, err := db.Query(FindMutationsSql,
		sql.Named("chr", location.BaseChr()),
		sql.Named("start", location.Start()),
		sql.Named("end", location.End()))

	if err != nil {
		return nil, err
	}

	mutations, err := rowsToMutations(rows)

	if err != nil {
		return nil, err
	}

	return &DatasetResults{Dataset: dataset.PublicId, Mutations: mutations}, nil
}

func GetPileup(search *SearchResults) (*PileupResults, error) {
	// first lets fix deletions and insertions
	for _, datasetResults := range search.DatasetResults {
		for _, mutation := range datasetResults.Mutations {
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
	}

	// put together by position, type, tum

	pileupMap := make(map[int]map[string]map[string][]*Mutation)

	for _, datasetResults := range search.DatasetResults {
		for _, mutation := range datasetResults.Mutations {

			mutation.Dataset = datasetResults.Dataset

			switch mutation.Type {
			case "3:DEL":
				for i := range mutation.End - mutation.Start + 1 {
					addToPileupMap(&pileupMap, mutation.Start+i, mutation)
				}
			case "2:INS":
				addToPileupMap(&pileupMap, mutation.Start, mutation)
			default:
				// deal with concatenated snps
				//tum := []rune(mutation.Tum)
				for i, c := range mutation.Tum {
					// clone and change tumor
					mut2 := mutation.Clone()
					mut2.Tum = string(c)
					addToPileupMap(&pileupMap, mut2.Start+int(i), mut2)
				}
			}
		}
	}

	location := search.Location

	// init pileup
	pileup := make([][]*Mutation, location.Len())

	for i := range location.Len() {
		pileup[i] = []*Mutation{}
	}

	// get sorted start positions
	starts := make([]int, 0, len(pileupMap))

	for start := range pileupMap {
		starts = append(starts, start)
	}

	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })

	// assemble pileups on each start location
	for _, start := range starts {
		// sort variant types
		variantTypes := make([]string, 0, len(pileupMap[start]))

		for variantType := range pileupMap[start] {
			variantTypes = append(variantTypes, variantType)
		}

		sort.Strings(variantTypes)

		for _, variantType := range variantTypes {
			// sort variant change
			tumors := make([]string, 0, len(pileupMap[start][variantType]))

			for tumor := range pileupMap[start][variantType] {
				tumors = append(tumors, tumor)
			}

			sort.Strings(tumors)

			for _, tumor := range tumors {
				mutations := pileupMap[start][variantType][tumor]

				for _, mutation := range mutations {
					offset := start - location.Start()

					pileup[offset] = append(pileup[offset], mutation)
				}

			}

		}
	}

	// extract the datasets on which dataframe we are using
	datasets := make([]string, 0, len(search.DatasetResults))

	for _, results := range search.DatasetResults {
		datasets = append(datasets, results.Dataset)
	}

	log.Debug().Msgf("what the %d", len(datasets))

	return &PileupResults{Location: location, Datasets: datasets, Pileup: pileup}, nil
}

func addToPileupMap(pileupMap *map[int]map[string]map[string][]*Mutation, start int, mutation *Mutation) {

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
		(*pileupMap)[start][mutation.Type][mutation.Tum] = make([]*Mutation, 0, 100)
	}

	(*pileupMap)[start][mutation.Type][mutation.Tum] = append((*pileupMap)[start][mutation.Type][mutation.Tum], mutation)
}

func rowsToMutations(rows *sql.Rows) ([]*Mutation, error) {

	mutations := make([]*Mutation, 0, 100)

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
			&mutation.Sample)

		if err != nil {
			fmt.Println(err)
		}

		mutations = append(mutations, &mutation)
	}

	log.Debug().Msgf("all the muts %d", len(mutations))

	return mutations, nil
}

func NewMutationDBCache(dir string) *DatasetCache {

	cacheMap := make(map[string]map[string]*Dataset)

	log.Debug().Msgf("---- mutations ----")

	assemblyFiles, err := os.ReadDir(dir)

	if err != nil {
		log.Fatal().Msgf("%s", err)

	}

	for _, assemblyDir := range assemblyFiles {

		if !assemblyDir.IsDir() {
			continue
		}

		dbFiles, err := os.ReadDir(filepath.Join(dir, assemblyDir.Name()))

		if err != nil {
			log.Fatal().Msgf("%s", err)

		}

		// init the cache
		//cacheMap[assemblyFile.Name()] = make(map[string]*MutationDB)

		for _, dbFile := range dbFiles {
			if !strings.HasSuffix(dbFile.Name(), ".db") {
				continue
			}

			path := filepath.Join(dir, assemblyDir.Name(), dbFile.Name())

			log.Debug().Msgf("Loading mutations from %s...", path)

			//metadata := NewMutationDBMetaData(assemblyFile.Name(), dbFile.Name())

			dataset, err := NewDataset(path)

			if err != nil {
				log.Fatal().Msgf("%s", err)
			}

			log.Debug().Msgf("Caching %s", dataset.PublicId)

			_, ok := cacheMap[dataset.Assembly]

			if !ok {
				cacheMap[dataset.Assembly] = make(map[string]*Dataset)
			}

			//cacheMap[dataset.Assembly][dataset.ShortName] = dataset
			cacheMap[dataset.Assembly][dataset.PublicId] = dataset
		}
	}

	log.Debug().Msgf("---- end ----")

	return &DatasetCache{dir: dir, cacheMap: cacheMap}
}

func (cache *DatasetCache) Dir() string {
	return cache.dir
}

func (cache *DatasetCache) ListDatasets(assembly string) ([]*Dataset, error) {

	cacheMap, ok := cache.cacheMap[assembly]

	if !ok {
		// assembly doesn't exist, so return empty array
		return []*Dataset{}, nil
	}

	ret := make([]*Dataset, 0, len(cacheMap))

	ids := make([]string, 0, len(cacheMap))

	for id := range cacheMap {
		ids = append(ids, id)
	}

	sort.Strings(ids)
	var dataset *Dataset

	for _, id := range ids {
		dataset = cacheMap[id]

		if dataset.Assembly == assembly {
			ret = append(ret, cacheMap[id])
		}
	}

	slices.SortFunc(ret,
		func(a, b *Dataset) int {
			return strings.Compare(a.Name, b.Name)
		},
	)

	return ret, nil
}

func (cache *DatasetCache) GetDataset(assembly string, publicId string) (*Dataset, error) {
	dataset, ok := cache.cacheMap[assembly][publicId]

	if !ok {
		return nil, fmt.Errorf("dataset not found")
	}

	return dataset, nil
}

func (cache *DatasetCache) Search(assembly string, location *dna.Location, publicIds []string) (*SearchResults, error) {
	results := SearchResults{Location: location, DatasetResults: make([]*DatasetResults, 0, len(publicIds))}

	for _, publicId := range publicIds {
		dataset, err := cache.GetDataset(assembly, publicId)

		if err != nil {
			return nil, err
		}

		datasetResults, err := dataset.Search(location)

		if err != nil {
			return nil, err
		}

		results.DatasetResults = append(results.DatasetResults, datasetResults)
	}

	return &results, nil
}
