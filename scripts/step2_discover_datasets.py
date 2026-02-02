# -*- coding: utf-8 -*-
"""
Encode read counts per base in 2 bytes

@author: Antony Holmes
"""
import argparse
import os
import sqlite3

import uuid_utils as uuid

rdf_view_id = "019c1f92-5632-71cb-a9de-487b80d376a5"

dir = f"../data/modules/mutations/"

genome_map = {
    "hg19": "Human",
    "hg38": "Human",
    "grch38": "Human",
    "mm10": "Mouse",
    "rn6": "Rat",
}


data = []

datasets = {}

for root, dirs, files in os.walk(dir):
    if "trash" in root:
        continue

    for filename in files:
        if "dataset.db" not in filename:
            continue

        relative_dir = root.replace(dir, "")

        print(relative_dir, filename)

        assembly, dataset_name = relative_dir.split("/")

        genome = genome_map.get(assembly.lower(), assembly)

        dataset_name = dataset_name.replace("_", " ")

        if dataset_name not in datasets:
            dataset_id = uuid.uuid7()
            datasets[dataset_name] = {
                "id": dataset_id,
                "name": dataset_name,
                "genome": genome,
                "assembly": assembly,
            }

        dataset = datasets[dataset_name]

        # filepath = os.path.join(root, filename)
        print(root, filename, relative_dir, genome, dataset)

        conn = sqlite3.connect(os.path.join(root, filename))
        conn.row_factory = sqlite3.Row

        # Create a cursor object
        cursor = conn.cursor()

        # Execute a query to fetch data
        cursor.execute(
            "SELECT id, genome, assembly, name, short_name, mutations, description FROM dataset"
        )

        # Fetch all results
        results = cursor.fetchall()

        # Print the results
        for row in results:
            row = {
                "id": row["id"],
                "genome": row["genome"],
                "assembly": row["assembly"],
                "name": row["name"],
                "short_name": (
                    dataset["short_name"]
                    if "short_name" in dataset
                    else dataset["name"].replace(" ", "_")
                ),
                "mutations": row["mutations"],
                "description": row["description"],
                "url": os.path.join(relative_dir, filename),  # where to find the sql db
            }
            # row.append(generate("0123456789abcdefghijklmnopqrstuvwxyz", 12))
            # row.append(dataset["id"])
            # row.append("Seq")
            # row.append(relative_dir)
            # row.append(dataset)
            data.append(row)

        conn.close()

db = os.path.join(dir, "datasets.db")

print(db)

if os.path.exists(db):
    os.remove(db)

conn = sqlite3.connect(db)
conn.row_factory = sqlite3.Row
cursor = conn.cursor()

cursor.execute("BEGIN TRANSACTION;")

cursor.execute(
    f"""CREATE TABLE permissions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
    );
"""
)

cursor.execute(
    f"INSERT INTO permissions (id, name) VALUES ('{rdf_view_id}', 'rdf:view');",
)


cursor.execute(
    f"""CREATE TABLE datasets (
    id TEXT PRIMARY KEY,
    genome TEXT NOT NULL,
    assembly TEXT NOT NULL,
    name TEXT NOT NULL,
    short_name TEXT NOT NULL,
    mutations INTEGER NOT NULL DEFAULT -1,
    description TEXT NOT NULL DEFAULT "",
    url TEXT NOT NULL
    );
    """
)

cursor.execute(
    f"""CREATE TABLE dataset_permissions (
    dataset_id TEXT NOT NULL,
    permission_id TEXT NOT NULL,
    PRIMARY KEY (dataset_id, permission_id),
    FOREIGN KEY(dataset_id) REFERENCES datasets(id),
    FOREIGN KEY(permission_id) REFERENCES permissions(id)
    );
"""
)


cursor.execute("COMMIT;")

cursor.execute("BEGIN TRANSACTION;")

for dataset in data:
    cursor.execute(
        f"""INSERT INTO datasets (id, genome, assembly, name, short_name, mutations, description, url) VALUES (
            '{dataset["id"]}',
            '{dataset["genome"]}',
            '{dataset["assembly"]}',
            '{dataset["name"]}',
            '{dataset["short_name"]}',
            {dataset["mutations"]},
            '{dataset["description"]}',
            '{dataset["url"]}');
        """,
    )

    cursor.execute(
        f"INSERT INTO dataset_permissions (dataset_id, permission_id) VALUES ('{dataset["id"]}', '{rdf_view_id}');",
    )

cursor.execute("COMMIT;")
