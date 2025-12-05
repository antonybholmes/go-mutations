import collections
import os
import sqlite3
import sys
import pandas as pd
from nanoid import generate
from uuid_utils import uuid7

idMap = {"20_icg": "20icg", "29_cell_lines": "29cl", "73_bcca": "73primary"}

renameMap = {
    "20_icg": "20 ICG",
    "29_cell_lines": "29 cell lines",
    "73_bcca": "73 primary",
}

dir = f"../data/modules/mutations/hg19/"

file = "/ifs/archive/cancer/Lab_RDF/scratch_Lab_RDF/ngs/wgs/data/human/rdf/hg19/mutation_database/93primary_29cl_dlbcl_hg19/samples.txt"

df_samples = pd.read_csv(file, sep="\t", header=0, keep_default_na=False)
datasets = list(sorted(df_samples["Dataset"].unique()))
sampleMap = {sample: uuid7() for sample in df_samples["Sample"].values}
sampleIdMap = {sample: i for i, sample in enumerate(df_samples["Sample"].values)}


print(sampleMap)


file = "/ifs/archive/cancer/Lab_RDF/scratch_Lab_RDF/ngs/wgs/data/human/rdf/hg19/mutation_database/93primary_29cl_dlbcl_hg19/93primary_29cl_rename_samples_hg19.maf.txt"

df = pd.read_csv(file, sep="\t", header=0, keep_default_na=False)

for dataset in datasets:
    dataset_uuid = uuid7()  # generate("0123456789abcdefghijklmnopqrstuvwxyz", 12)
    shortName = idMap[dataset]
    df_samples_d = df_samples[df_samples["Dataset"] == dataset]

    print(df_samples_d["Sample"].shape)

    dfd = df[df["Sample"].isin(df_samples_d["Sample"])]

    db = os.path.join(dir, f"{shortName}.db")

    print(db)

    if os.path.exists(db):
        os.remove(db)

    conn = sqlite3.connect(db)
    cursor = conn.cursor()

    cursor.execute("BEGIN TRANSACTION;")

    cursor.execute(
        f"""CREATE TABLE info (
        id TEXT PRIMARY KEY ASC,
        short_name TEXT NOT NULL,
        name TEXT NOT NULL,
        description TEXT NOT NULL DEFAULT "",
        assembly TEXT NOT NULL
        );
    """
    )

    cursor.execute(
        f""" CREATE TABLE samples (
        id TEXT PRIMARY KEY ASC,
        name TEXT NOT NULL,
        coo TEXT NOT NULL,
        lymphgen TEXT NOT NULL,
        paired_normal_dna INTEGER NOT NULL,
        institution TEXT NOT NULL,
        sample_type TEXT NOT NULL
        );
    """
    )

    cursor.execute(
        f""" CREATE TABLE mutations (
        id TEXT PRIMARY KEY ASC,
        sample_id TEXT NOT NULL,
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
        FOREIGN KEY(sample_id) REFERENCES samples(id)
        );
        """
    )

    cursor.execute("COMMIT;")

    name = renameMap[dataset]

    cursor.execute("BEGIN TRANSACTION;")

    cursor.execute(
        f'INSERT INTO info (id, short_name, name, assembly) VALUES ("{dataset_uuid}", "{shortName}", "{name}", "hg19");',
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
            f"INSERT INTO samples (id, name, coo, lymphgen, paired_normal_dna, institution, sample_type) VALUES ('{sampleMap[sample]}', '{sample}', '{coo}', '{lymphgen}', {paired}, '{ins}', '{sample_type}');",
        )

    cursor.execute("COMMIT;")

    cursor.execute("BEGIN TRANSACTION;")
    for i in range(dfd.shape[0]):
        mutation_uuid = uuid7()  # generate("0123456789abcdefghijklmnopqrstuvwxyz", 12)

        chr = dfd["Chromosome"].values[i]
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
        sample_uuid = sampleMap[sample]
        # sample_id = sampleIdMap[sample]

        t_alt_count = dfd["t_alt_count"].values[i]
        t_depth = dfd["t_depth"].values[i]

        if t_alt_count == "na":
            t_alt_count = -1

        if t_depth == "na":
            t_depth = -1

        # so we can merge mutations from different tables, use the public_id as foreign key
        cursor.execute(
            f"INSERT INTO mutations (id, sample_id, chr, start, end, ref, tum, t_alt_count, t_depth, variant_type, vaf) VALUES ('{mutation_uuid}', '{sample_uuid}', '{chr}', {start}, {end}, '{ref}', '{tum}', {t_alt_count}, {t_depth}, '{variant_type}', {vaf});",
        )

    cursor.execute("COMMIT;")

    cursor.execute("BEGIN TRANSACTION;")

    cursor.execute(
        f"""CREATE INDEX mutations_chr_start_end_idx ON mutations (chr, start, end);"""
    )
    cursor.execute(
        f"""CREATE INDEX mutations_gene_idx ON mutations (hugo_gene_symbol); """
    )
    cursor.execute("COMMIT;")
