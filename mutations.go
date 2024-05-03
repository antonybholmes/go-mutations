package mutations

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"

	"github.com/antonybholmes/go-dna"
	"github.com/antonybholmes/go-sys"
)

const ALL_PLATFORMS_SQL = `SELECT uuid, name
	FROM platforms ORDER BY name`

const PLATFORM_SAMPLES_SQL = `SELECT uuid, name
	FROM samples WHERE platform = ?1 ORDER BY name`

const FIND_SAMPLES_SQL = `SELECT uuid, array, name
	FROM samples WHERE platform = ?1 AND name LIKE ?2 ORDER BY array, name`

type MutationsDB struct {
	Path             string
	Db               *sql.DB
	AllPlatformsStmt *sql.Stmt
	AllSamplesStmt   *sql.Stmt
	FindSamplesStmt  *sql.Stmt
}

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

type MutationDB struct {
	Uuid     string `json:"uuid"`
	Name     string `json:"name"`
	Assembly string `json:"assembly"`
}

type Mutation struct {
	Uuid   string  `json:"uuid"`
	Chr    string  `json:"crh"`
	Start  uint32  `json:"start"`
	End    uint32  `json:"end"`
	Ref    string  `json:"ref"`
	Mut    string  `json:"mut"`
	Vaf    float32 `json:"vaf"`
	Sample string  `json:"sample"`
	Gene   string  `json:"gene"`
}

type MutationResults struct {
	DB        *MutationDB `json:"db"`
	Mutations []Mutation  `json:"mutations"`
}

func NewMutationDB(dir string) (*MutationsDB, error) {
	db := sys.Must(sql.Open("sqlite3", path.Join(dir, "mutations.db")))

	return &MutationsDB{Db: db,
		Path:             dir,
		AllPlatformsStmt: sys.Must(db.Prepare(ALL_PLATFORMS_SQL)),
		AllSamplesStmt:   sys.Must(db.Prepare(PLATFORM_SAMPLES_SQL)),
		FindSamplesStmt:  sys.Must(db.Prepare(FIND_SAMPLES_SQL)),
	}, nil
}

func (mutationsdb *MutationsDB) AllPlatforms() (*[]MutationDB, error) {
	var uuid string
	var name string

	rows, err := mutationsdb.AllPlatformsStmt.Query()

	if err != nil {
		return nil, err
	}

	samples := []Mutation{}

	defer rows.Close()

	for rows.Next() {

		err := rows.Scan(&uuid, &array, &name)

		if err != nil {
			fmt.Println(err)
		}

		samples = append(samples, Mutation{uuid, name})
	}

	return &MutationResults{platform, samples}, nil
}

func (mutationsdb *MutationsDB) AllSamples() (*[]Mutation, error) {

	rows, err := mutationsdb.AllSamplesStmt.Query()

	if err != nil {
		return nil, err
	}

	return rowsToSamples(rows)
}

func (mutationsdb *MutationsDB) FindSamples(platform *MutationsDB, search string) (*MutationResults, error) {

	rows, err := mutationsdb.FindSamplesStmt.Query(platform.Uuid, fmt.Sprintf("%%%s%%", search))

	if err != nil {
		return nil, err
	}

	return rowsToSamples(platform, rows)
}

func (mutationsdb *MutationsDB) Expression(samples *MutationsReq) (*ExpressionData, error) {
	// let sample_ids = vec![
	//     "0c3b8a19-1975-4c6e-aece-44a59c71719d",
	//     "0c4f0c89-af16-484a-a408-8dfde25d8f10",
	// ];

	nSamples := len(samples.Samples)

	//eprintln!("{:?}", Path::new(&self.path).join("meta.tsv").to_str());
	// open meta data
	file, err := os.Open(path.Join(mutationsdb.Path, samples.Platform, "meta.tsv"))

	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(file)
	reader.Comma = '\t'

	columnValues, err := reader.Read()

	if err != nil {
		return nil, err
	}

	probeIds := columnValues[1:]

	nProbes := len(probeIds)

	columnValues, err = reader.Read()

	if err != nil {
		return nil, err
	}

	entrezIds := columnValues[1:]

	columnValues, err = reader.Read()

	if err != nil {
		return nil, err
	}

	geneSymbols := columnValues[1:]

	rowRecords := make([][]float64, nSamples)
	sampleNames := make([]string, nSamples)

	// let mut rdr = csv::ReaderBuilder::new()
	//     .has_headers(true)
	//     .delimiter(b'\t')
	//     .from_reader(file);

	for _, sample := range samples.Samples {
		file, err := os.Open(path.Join(mutationsdb.Path, fmt.Sprintf("%s.tsv", sample)))

		if err != nil {
			return nil, err
		}

		reader := csv.NewReader(file)
		reader.Comma = '\t'

		columnValues, err := reader.Read()

		if err != nil {
			return nil, err
		}

		sampleNames = append(sampleNames, columnValues[0])

		for {
			columnValues, err := reader.Read()

			if err == io.EOF {
				break
			}

			if err != nil {
				return nil, err
			}

			rowRecords = append(rowRecords, sys.Map(columnValues[1:], func(s string) float64 {
				v, err := strconv.ParseFloat(s, 64)

				if err != nil {
					return 0
				}

				return v
			}))
		}
	}

	data := ExpressionData{
		Exp:    make([][]float64, nProbes),
		Header: sampleNames,
		Index: ExpressionDataIndex{
			ProbeIds:    probeIds,
			EntrezIds:   entrezIds,
			GeneSymbols: geneSymbols,
		},
	}

	// We are going to transpose the data we read into this output
	// array
	//let mut data:Vec<Vec<f64>> = vec![vec![0.0; n_samples]; n_probes] ;

	for row := range nProbes {
		data.Exp[row] = make([]float64, nSamples)

		for col := range nSamples {
			data.Exp[row][col] = rowRecords[col][row]
		}

		//eprintln!("{:?} {} out_row", out_row, n_samples);

		// wtr.write_record(&out_row)?;
	}

	// for result in rdr.records() {
	//     //for row in data {
	//     let record =
	//         result.map_err(|_| MicroarrayError::FileError("header issue".to_string()))?;

	//     let row = cols.iter().map(|c| &record[**c]).collect::<Vec<&str>>();
	//     //println!("{}", &record[0]);
	//     wtr.write_record(&row)
	//         .map_err(|_| MicroarrayError::FileError("header issue".to_string()))?;
	// }

	//let vec: Vec<u8> = wtr.into_inner()?;

	//let data = String::from_utf8(vec)?;

	return &data, nil
}

func (microarraydb *MutationDB) Close() {
	microarraydb.Db.Close()
}

func rowsToSamples(platform *MutationsDB, rows *sql.Rows) (*MutationResults, error) {
	var uuid string
	var array string
	var name string

	samples := []Mutation{}

	defer rows.Close()

	for rows.Next() {

		err := rows.Scan(&uuid, &array, &name)

		if err != nil {
			fmt.Println(err)
		}

		samples = append(samples, Mutation{uuid, name})
	}

	return &MutationResults{platform, samples}, nil
}
