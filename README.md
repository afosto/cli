# Afosto CLI

This tool is currently under heavy development, use with caution.

## Installing

To install the CLI tool there are several options. There are no dependencies.

### MacOS install / update

To install use [homebrew](https://brew.sh) directly from the [Afosto tap](https://github.com/afosto/homebrew-tap).

```bash
brew tap afosto/tap
brew install afosto-cli
```

To upgrade:

```bash
brew upgrade afosto-cli
```

## Features

The current list for features for this CLI tool is as follows:

- upload a local directory to your account
- download a remote directory (from your account) to your machine
- develop HTML templates 

To start local development of templates, run: 

## Upload files

In order to upload a directory and all it's contents from your machine to your account use the following command:
```bash
afosto upload -s /Users/peter/images -d /images
```
`-s` (source) points to the path on your computer that you want to recursively upload. `-d` (destination) points to the directory in your account.

## Download files

When you want to download files from your account to your computer you run:

```bash
afosto download -s invoices -d /Users/peter/backups/invoices
```

`-s` (source) points to the directory in your account. `-d` (destination) points to the path on your computer that you want to store the files.


## Develop templates

To start working on templates in your account you need to start the local development server while pointing to your configuration file. 

```bash
afosto render -p 8888 -f afosto.config.yml
```

Where `-p` is the port number for the local webserver and `-f` points to the afosto.config.yml file in the root of the template repository.

### Templating language

Within the templates that are rendered there a some (hidden) powerful features / filters that might be of use. 

#### Sort

Sort takes in a slice or a map and sort this based on a key. Start with `-` in order to sort in ascending order.
```
{{slice|sort}}
{{slice|sort:"filters.key"}}
{{slice|sort:"-filters.key"}}
``` 

 
 