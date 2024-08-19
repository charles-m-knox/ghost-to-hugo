# Simple example

This example connects to a mysql database and puts markdown files into the specified directory.

## Usage

```bash
git clone https://git.cmcode.dev/cmcode/ghost-to-hugo.git
cd ghost-to-hugo
cp config.example.json config.json

# now, edit config.json to meet your needs

go get -v
go build -v
./ghost-to-hugo -f config.json

# done
```

Depending on your Hugo application's configuration/theme/etc, you will likely need to change the default template. This is a little tricky because of JSON's syntax, but the `config.example.json` file demonstrates what a valid template looks like.

## Tips for connecting to a remote mysql db

If your mysql database is only accessible behind an ssh tunnel, you can use ssh forwarding:

```bash
ssh -L 3306:127.0.0.1:3306 user@server
```
