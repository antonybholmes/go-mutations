PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

 
CREATE TABLE info (
	id INTEGER PRIMARY KEY ASC,
	public_id TEXT NOT NULL,
	short_name TEXT NOT NULL,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT "",
	assembly TEXT NOT NULL);

 
CREATE TABLE samples (
	id INTEGER PRIMARY KEY ASC,
	public_id TEXT NOT NULL,
	name TEXT NOT NULL,
	coo TEXT NOT NULL,
	lymphgen TEXT NOT NULL,
	paired_normal_dna INTEGER NOT NULL,
	institution TEXT NOT NULL,
	sample_type TEXT NOT NULL);

CREATE TABLE mutations (
	id INTEGER PRIMARY KEY ASC,
	sample_public_id TEXT NOT NULL,
	hugo_gene_symbol TEXT NOT NULL DEFAULT '',
	variant_classification TEXT NOT NULL DEFAULT '',
	variant_type TEXT NOT NULL DEFAULT '',
	chr TEXT NOT NULL,
	start INTEGER NOT NULL,
	end INTEGER NOT NULL,
	ref TEXT NOT NULL,
	tum TEXT NOT NULL,
	t_alt_count INTEGER NOT NULL DEFAULT -1,
	t_depth INTEGER NOT NULL DEFAULT -1,
	vaf FLOAT NOT NULL DEFAULT -1,
	FOREIGN KEY(sample_public_id) REFERENCES samples(public_id));
CREATE INDEX mutations_chr_start_end_idx ON mutations (chr, start, end);
CREATE INDEX mutations_gene_idx ON mutations (hugo_gene_symbol); 
