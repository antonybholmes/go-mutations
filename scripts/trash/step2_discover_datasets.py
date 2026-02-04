# -*- coding: utf-8 -*-
"""
Encode read counts per base in 2 bytes

@author: Antony Holmes
"""
import argparse
import collections
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


samples = collections.defaultdict(list)

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

        # filepath = os.path.join(root, filename)
        print(root, filename, relative_dir, genome, dataset_name)

        conn = sqlite3.connect(os.path.join(root, filename))
        conn.row_factory = sqlite3.Row

        # Create a cursor object
        cursor = conn.cursor()

        # Execute a query to fetch data
        cursor.execute(
            "SELECT id, genome, assembly, name, short_name, mutations, description FROM dataset"
        )

        # Fetch all results
        results = cursor.fetchone()

        url = os.path.join(relative_dir, filename)

        # Print the results

        row = {
            "id": results["id"],
            "genome": results["genome"],
            "assembly": results["assembly"],
            "name": results["name"],
            "short_name": (
                results["short_name"]
                if "short_name" in results
                else results["name"].replace(" ", "_")
            ),
            "mutations": results["mutations"],
            "description": results["description"],
            "url": url,  # where to find the sql db
        }

        data.append(row)

        cursor.execute("SELECT id, name FROM samples")

        rows = cursor.fetchall()

        for row in rows:
            samples[results["id"]].append(
                {
                    "id": row["id"],
                    "name": row["name"],
                    "url": url,
                    "dataset_id": results["id"],
                    # where to find the sql db
                }
            )

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
    f"""CREATE TABLE samples (
    id TEXT PRIMARY KEY,
    dataset_id TEXT NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    FOREIGN KEY(dataset_id) REFERENCES datasets(id)
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

    for sample in samples[dataset["id"]]:
        cursor.execute(
            f"""INSERT INTO samples (id, dataset_id, name, url) VALUES (
                '{sample["id"]}',
                '{dataset["id"]}',
                '{sample["name"]}',
                '{sample["url"]}');
            """,
        )

cursor.execute("COMMIT;")
