# remove any db.json already present
rm db.json;

# build binary
go build -o out;

# run that shiz
./out