# ghost-to-hugo

Takes Ghost blog posts and converts them to Hugo-compatible markdown files.

## Setup

If your mysql database is only accessible behind an ssh tunnel, you can use ssh forwarding:

```bash
ssh -L 3306:127.0.0.1:3306 user@server
```
