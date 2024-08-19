# ghost-to-hugo

Takes Ghost blog posts and converts them to Hugo-compatible markdown files.

This leverages a database connection - Ghost recommends mysql as its supported database type, but it can technically work with any compliant driver that implements Go's `database/sql` interfaces.

This is compatible with **Ghost v5.87.1**. Compatibility with any other version is not guaranteed.

## Example Usage

See [`examples/simple/README.md`](./examples/simple/README.md).

## Warnings

**Please verify the output of this application before publishing it**. If your configuration is incorrect, it may publish premium content from your Ghost instance to the world, or it may publish newsletters as posts.

## Motivation

Ghost is an incredible project that has empowered myself and others in my community to achieve higher productivity with digital content marketing online.

However, Ghost is very JavaScript-heavy. In my browser, Ghost admin/blog post pages can use up to 200MB RAM per tab. I personally like to use less RAM if possible.

I wanted an option to use both Ghost and statically rendered Markdown files that work with Hugo. A browser tab using a barebones Hugo theme can use between 20-30MB RAM, almost 10% of what the Ghost-equivalent uses.

The idea is to run `ghost-to-hugo` on a regular basis, such as once per day or hour, and then serve the Hugo-rendered content from e.g. `https://nojs.example.com`, whilst simultaneously serving your Ghost-powered content at `https://example.com`.

## Structure

In order to keep dependencies to zero (see [`go.mod`](./go.mod) - it only uses the standard library), this Go module is structured as a library that can be imported by any application.

This has the benefit of not requiring any specific SQL driver - it accepts SQL rows themselves from the `database/sql` package. You can use any compliant driver, such as sqlite or postgres - although I haven't tried anything aside from mysql, so tread carefully.

## Other notes

There may be other hidden capabilities in this that I've not documented or explored fully. For example, it *may* be possible to get creative with templates to support rendering the featured image for a post (the default template does not support this currently).

`.json` files are used for configuration because I do not want to pull in larger dependencies like `yaml`. `xml` could be supported in the future (because it's included in the standard library) - it's a bit more desirable than JSON because templates can span multiple lines, and that gets a little awkward with JSON syntax.

## Development notes

The `ghosttohugo` package is the primary library for this module. Its unit test coverage is currently at **`70.7%`** and at this time, I do not intend to get it higher. The core business logic is well-tested; everything else remaining is not worth unit testing.

## Compatibility

Here is a list of all tested Ghost versions that are known to work:

- v5.87.1

The only table that needs to be read from the database is the `posts` table. If any database migrations occur upstream, this application will likely break.
