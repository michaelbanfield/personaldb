PersonalDB
===========================

A serverless personal SQLite database accessed over HTTP.

Based on [litestream-cloud-run-example](https://github.com/steren/litestream-cloud-run-example)

Components of this service are

* A minimal go http server that sends queries to SQLite
* Google Cloud Run - used to provide serverless compute, scales to 0 within 15 minutes of your last query
* Litestream - used to sync the state of the database to GCS object storage - no expensive file system required


## Why?

I wanted a simple way to store personal data (notes, things I would normally put in a spreadsheet), and query it using SQL.

Most cheap/free options with public clouds are either not relational databases (e.g. DynamoDB) or have a short free trial (fundamentally beause they use expensive file systems).

Running a database server on a VM or locally is more expensive, and I don't want to manage it. Additionally connecting to these databases requires using some sort of sql client, which is not ideal for a mobile device.

This project uses a SQLite database, which is a single file, and syncs it to Google Cloud Storage using Litestream. It exposes a HTTP API, so if you have curl, you can use it.

## Usage

### Prerequisites

* [Google Cloud](https://console.cloud.google.com/) Account and project 
* Pick a region based on [GCPPing](https://gcping.com/)
* [Create a Cloud Storage bucket](https://cloud.google.com/storage/docs/creating-buckets) in a the same region
* Install the google cloud cli

### Build & deploy the sample to Cloud Run

Note: If you dont have a development environment all these steps can be done in the Google Cloud Shell (and its even a bit easier as the cli is pre installed).

Clone this repository and navigate to the cloned location.

```sh
git clone git@github.com:michaelbanfield/personaldb.git
cd personaldb
```

Then build and deploy the application with the following command:

```sh
gcloud beta run deploy personaldb \
  --source .  \
  --set-env-vars REPLICA_URL=gcs://BUCKET_NAME/database \
  --max-instances 1 \
  --execution-environment gen2 \
  --no-cpu-throttling \
  --region REGION \
  --project PROJECT_ID
```

Replace:

* `BUCKET_NAME` with your Cloud Storage bucket name
* `REGION` with the same region where you created the bucket, for example `us-central1`
* `PROJECT_ID` with your Google Cloud project ID.

When the deployment completes, take note of `.run.app` URL of the Cloud Run service. and run the following.

```sh
alias personaldb="curl -H 'Authorization: Bearer $(gcloud auth print-identity-token)' https://<YOUR_URL>.run.app/query -d"
personaldb "CREATE TABLE notes (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP
);"
personaldb "INSERT INTO notes (title, body) VALUES ('Hello', 'World');"
personaldb "SELECT * FROM notes;"
```

### Security

The cloud run configuration above requires authentication to access the service. This is done using a bearer token, which is generated using the `gcloud auth print-identity-token` command.

You can follow the instructions to add extra users [here](https://cloud.google.com/run/docs/securing/managing-access).

It is not recommended to expose this service publically.


## Development

The http server can be run locally with

```sh
go run main.go --dsn=database.db
```

Which will just create an empty database.