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

## Usage

To start local development of templates, run: 

```bash
afosto render -p 8888 -f afosto.config.yml
```

Where `-p` is the port number for the local webserver and `-f` points to the afosto.config.yml file in the root of the template repository.





## Filters

Sort

sort takes in a slice or a map and sort this based on a key.

to determine if it will be asc or desc, first parameter can be a `-` which controls the order of the sorting
after that it takes in a dotted path. which represents the nesting it should check
```

{{slice|sort}}

{{slice|sort:"filters.key"}}
 
{{slice|sort:"-filters.key"}}

``` 

 
 