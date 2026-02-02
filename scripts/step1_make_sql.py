import collections
import os
import sqlite3
import sys

import pandas as pd
from nanoid import generate
from uuid_utils import uuid7

genome = "Human"
assembly = "hg19"

idMap = {"20_icg": "20icg", "29_cell_lines": "29cl", "73_bcca": "73primary"}

renameMap = {
    "20_icg": "20 ICG",
    "29_cell_lines": "29 cell lines",
    "73_bcca": "73 primary",
}

HUMAN_CHRS = [
    "chr1",
    "chr2",
    "chr3",
    "chr4",
    "chr5",
    "chr6",
    "chr7",
    "chr8",
    "chr9",
    "chr10",
    "chr11",
    "chr12",
    "chr13",
    "chr14",
    "chr15",
    "chr16",
    "chr17",
    "chr18",
    "chr19",
    "chr20",
    "chr21",
    "chr22",
    "chrX",
    "chrY",
    "chrM",
]

CHR_MAP = {chr: idx + 1 for idx, chr in enumerate(HUMAN_CHRS)}

MOUSE_CHRS = [
    "chr1",
    "chr2",
    "chr3",
    "chr4",
    "chr5",
    "chr6",
    "chr7",
    "chr8",
    "chr9",
    "chr10",
    "chr11",
    "chr12",
    "chr13",
    "chr14",
    "chr15",
    "chr16",
    "chr17",
    "chr18",
    "chr19",
    "chrX",
    "chrY",
    "chrM",
]

dir = f"../data/modules/mutations/hg19/"

file = "/ifs/archive/cancer/Lab_RDF/scratch_Lab_RDF/ngs/wgs/data/human/rdf/hg19/mutation_database/93primary_29cl_dlbcl_hg19/samples.txt"

df_samples = pd.read_csv(file, sep="\t", header=0, keep_default_na=False)
datasets = list(sorted(df_samples["Dataset"].unique()))
sample_map = {sample: uuid7() for sample in df_samples["Sample"].values}
# sampleIdMap = {sample: i for i, sample in enumerate(df_samples["Sample"].values)}

metadata = ["COO", "LymphGen class", "Paired normal DNA", "Institution", "Sample type"]

chrs = HUMAN_CHRS if genome.lower() == "human" else MOUSE_CHRS
chr_map = {chr: idx + 1 for idx, chr in enumerate(chrs)}

metadata_map = {meta: mi + 1 for mi, meta in enumerate(metadata)}


print(sample_map)


file = "/ifs/archive/cancer/Lab_RDF/scratch_Lab_RDF/ngs/wgs/data/human/rdf/hg19/mutation_database/93primary_29cl_dlbcl_hg19/93primary_29cl_rename_samples_hg19.maf.txt"

df = pd.read_csv(file, sep="\t", header=0, keep_default_na=False)

# sort by Chromosome col
# df = df.sort_values(by="Chromosome")

for dataset in datasets:
    dataset_id = uuid7()  # generate("0123456789abcdefghijklmnopqrstuvwxyz", 12)
    shortName = idMap[dataset]
    df_samples_d = df_samples[df_samples["Dataset"] == dataset]

    print(df_samples_d["Sample"].shape)

    dfd = df[df["Sample"].isin(df_samples_d["Sample"])]

    mutation_counts = dfd.shape[0]

    dataset_dir = os.path.join(dir, shortName)

    if not os.path.exists(dataset_dir):
        os.makedirs(dataset_dir)

    db = os.path.join(dataset_dir, f"dataset.db")

    print(db)

    if os.path.exists(db):
        os.remove(db)

    conn = sqlite3.connect(db)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()

    cursor.execute("BEGIN TRANSACTION;")

    cursor.execute(
        f"""CREATE TABLE chromosomes (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL
        );
    """
    )

    for chr in chrs:
        cursor.execute(
            f"INSERT INTO chromosomes (id, name) VALUES ({chr_map[chr]}, '{chr}');"
        )
    cursor.execute("COMMIT;")

    cursor.execute("BEGIN TRANSACTION;")

    cursor.execute(
        f"""CREATE TABLE dataset (
        id TEXT PRIMARY KEY,
        genome TEXT NOT NULL,
        assembly TEXT NOT NULL,
        name TEXT NOT NULL,
        short_name TEXT NOT NULL,
        mutations INTEGER NOT NULL DEFAULT 0,
        description TEXT NOT NULL DEFAULT ""
        );
    """
    )

    cursor.execute(
        f""" CREATE TABLE metadata (
        id INTEGER PRIMARY KEY,
        name TEXT NOT NULL
        );
    """
    )

    cursor.execute(
        f""" CREATE TABLE samples (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL
        );
    """
    )

    cursor.execute(
        f""" CREATE TABLE sample_metadata (
        sample_id TEXT NOT NULL,
        metadata_id TEXT NOT NULL,
        value TEXT NOT NULL,
        PRIMARY KEY (sample_id, metadata_id),
        FOREIGN KEY(sample_id) REFERENCES samples(id),
        FOREIGN KEY(metadata_id) REFERENCES metadata(id)
        );
    """
    )

    cursor.execute(
        f""" CREATE TABLE mutations (
        id INTEGER PRIMARY KEY,
        sample_id TEXT NOT NULL,
        hugo_gene_symbol TEXT NOT NULL DEFAULT '',
        variant_classification TEXT NOT NULL DEFAULT '',
        variant_type TEXT NOT NULL DEFAULT '',
        chr_id INTEGER NOT NULL,
        start INTEGER NOT NULL,
        end INTEGER NOT NULL,
        ref TEXT NOT NULL,
        tum TEXT NOT NULL,
        t_alt_count INTEGER NOT NULL DEFAULT -1,
        t_depth INTEGER NOT NULL DEFAULT -1,
        vaf FLOAT NOT NULL DEFAULT -1,
        FOREIGN KEY(sample_id) REFERENCES samples(id),
        FOREIGN KEY(chr_id) REFERENCES chromosomes(id)
        );
        """
    )

    cursor.execute("COMMIT;")

    name = renameMap[dataset]

    cursor.execute("BEGIN TRANSACTION;")

    cursor.execute(
        f"INSERT INTO dataset (id, short_name, name, genome, assembly, mutations) VALUES ('{dataset_id}', '{shortName}', '{name}', '{genome}', '{assembly}', {mutation_counts});",
    )

    cursor.execute("COMMIT;")

    cursor.execute("BEGIN TRANSACTION;")

    for meta in metadata:
        cursor.execute(
            f"INSERT INTO metadata (id, name) VALUES ('{metadata_map[meta]}', '{meta}');",
        )

    cursor.execute("COMMIT;")

    cursor.execute("BEGIN TRANSACTION;")

    for i in range(df_samples_d.shape[0]):
        sample = df_samples_d["Sample"].values[i]
        coo = df_samples_d["COO"].values[i]

        if "nd" in coo.lower():
            coo = "NA"

        lymphgen = df_samples_d["LymphGen class"].values[i]
        paired = df_samples_d["Paired normal DNA"].values[i]
        ins = df_samples["Institution"].values[i]
        sample_type = df_samples_d["Sample type"].values[i]

        cursor.execute(
            f"INSERT INTO samples (id, name) VALUES ('{sample_map[sample]}', '{sample}');",
        )

        for meta in metadata:
            value = df_samples_d[meta].values[i]
            cursor.execute(
                f"INSERT INTO sample_metadata (sample_id, metadata_id, value) VALUES ('{sample_map[sample]}', {metadata_map[meta]}, '{value}');",
            )

    cursor.execute("COMMIT;")

    # cursor.close()

    # chrs = dfd["Chromosome"].unique()

    # for chr in chrs:
    #     dfd_chr = dfd[dfd["Chromosome"] == chr]

    #     db = f"mutations_{chr}.db"

    #     if os.path.exists(db):
    #         os.remove(db)

    #     conn = sqlite3.connect(db)
    #     cursor = conn.cursor()

    cursor.execute("BEGIN TRANSACTION;")

    for i in range(dfd.shape[0]):
        mutation_uuid = uuid7()  # generate("0123456789abcdefghijklmnopqrstuvwxyz", 12)

        chr = dfd["Chromosome"].values[i]

        chr = chr.replace("MT", "M")

        if chr not in chr_map:
            continue

        chr_id = chr_map[chr]

        # save space
        chr = chr.replace("chr", "")

        start = dfd["Start_Position"].values[i]
        end = dfd["End_Position"].values[i]
        ref = dfd["Reference_Allele"].values[i]
        tum = dfd["Tumor_Seq_Allele2"].values[i]
        vaf = dfd["VAF"].values[i]
        db = dfd["Database"].values[i]

        if vaf == "na":
            vaf = -1

        variant_type = dfd["Variant_Type"].values[i]
        sample = dfd["Sample"].values[i]
        sample_uuid = sample_map[sample]
        # sample_id = sampleIdMap[sample]

        t_alt_count = dfd["t_alt_count"].values[i]
        t_depth = dfd["t_depth"].values[i]

        if t_alt_count == "na":
            t_alt_count = -1

        if t_depth == "na":
            t_depth = -1

        # so we can merge mutations from different tables, use the public_id as foreign key
        cursor.execute(
            f"INSERT INTO mutations (sample_id, chr_id, start, end, ref, tum, t_alt_count, t_depth, variant_type, vaf) VALUES ('{sample_uuid}', {chr_id}, {start}, {end}, '{ref}', '{tum}', {t_alt_count}, {t_depth}, '{variant_type}', {vaf});",
        )

    cursor.execute("COMMIT;")

    cursor.execute("BEGIN TRANSACTION;")

    cursor.execute(
        f"""CREATE INDEX mutations_chr_id_start_end_idx ON mutations (chr_id, start, end);"""
    )
    cursor.execute(
        f"""CREATE INDEX mutations_gene_idx ON mutations (hugo_gene_symbol); """
    )
    cursor.execute("COMMIT;")

    # cursor.close()
    # conn.close()
