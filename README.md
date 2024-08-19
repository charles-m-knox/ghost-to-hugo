# ghost-to-hugo

Takes Ghost blog posts and converts them to Hugo-compatible markdown files.

## Setup

If your mysql database is only accessible behind an ssh tunnel, you can use ssh forwarding:

```bash
ssh -L 3306:127.0.0.1:3306 user@server
```

## Development notes

The `ghosttohugo` package is the primary library for this module. Its unit test coverage is currently at **`70.7%`** and at this time, I do not intend to get it higher. The core business logic is well-tested; everything else remaining is not worth unit testing.
