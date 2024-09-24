# Simple example

This example connects to a mysql database and puts markdown files into the specified directory.

## Usage

```bash
git clone https://github.com/charles-m-knox/ghost-to-hugo.git
cd examples/simple
cp config.example.json config.json

# now, edit config.json to meet your needs

go get -v
go build -v
./simple -f config.json

# done
```

Depending on your Hugo application's configuration/theme/etc, you will likely need to change the default template. This is a little tricky because of JSON's syntax, but the `config.example.json` file demonstrates what a valid template looks like.

## Tips for connecting to a remote mysql db

If your mysql database is only accessible behind an ssh tunnel, you can use ssh forwarding:

```bash
ssh -L 3306:127.0.0.1:3306 user@server
```

## Alternative: podman/docker container image

If you prefer not to build from source, you can use the pre-built container image.

You must first create a `config.json` just like above, and ensure that the output directory is going to exist within the container.

```bash
podman run --rm -it \
    -v "$(pwd)/config.json:/config.json:ro" \
    -v "/path/to/output:/path/to/output" \
    ghcr.io/charles-m-knox/ghost-to-hugo:simple-mysql
```

Note: If you're using an SSH port forwarding mechanism for the mysql database connection, you may want to consider adding `--network host` to the above `podman run` command.

### Building the container image

```bash
podman build \
    --build-arg GOSUMDB="${GOSUMDB}" \
    --build-arg GOPROXY="${GOPROXY}" \
    -f containerfile \
    -t ghcr.io/charles-m-knox/ghost-to-hugo:simple-mysql .
```
