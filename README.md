# klev examples in Golang

This repository contains a set of examples on how to interact with [klev](https://klev.dev)

## Running an example

To run the examples, you need to:
 * Register an account a [klev](https://dash.klev.dev)
 * Create a new token
 * Set this token as an env variable: `export KLEV_TOKEN_DEMO=<TOKEN>`
 * Build all examples: `make all`
 * Run a concrete example: `./bin/hello`

## Hello

This example:
 * creates a new log
 * posts a message to it
 * gets the message back and displays it

To run hello world: `./bin/hello`

## Files

This example uploads and downloads a file to a log.

```
# create a new file
$ echo "hello world" > in.txt

# upload the file
$ ./bin/files upload in.txt
Done. Upload log: 2I98sW36bR2oA8IOHEseEhwqVsd

# download the file, use the log above
$ ./bin/files download 2I98sW36bR2oA8IOHEseEhwqVsd out.txt

# compare the files
diff -s in.txt out.txt
```

## Chat

This example implements simple web chat server on top of klev.
 * to run the example: `./bin/chat`
 * to access the server: [http://127.0.0.1:8000](http://127.0.0.1:8000)

## Journal

Collects systemd journal logs and send them to klev log. Run:

```
$./bin/jounral
```
