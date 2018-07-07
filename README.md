# ISQool

*On-the-fly ISQ datasets for UNF courses*

---

Scrapes [historical course data](https://banner.unf.edu/pls/nfpo/wksfwbs.p_dept_schd) from UNF into a format suitable for visualization or analysis. Given a course code or professor's N#, this script will generate a CSV file from (and also cache in a local SQLite database) the following data: 
* ISQ rating distributions
* Grade distributions
* Course schedules

### Installation

This project requires **Go**. Their website provides [installers](https://golang.org/dl/). Mac and Linux users can alternatively use their respective package managers. Verify Go is properly installed by running `go env GOPATH`.

###### Quick install

```shell
# Install or update
$ go get -u github.com/rothso/isqool
```

### Usage

```shell
# Pull the data for Computer Science 1
$ isqool COP2220

# Pull the data for Ken Martin
$ isqool N00009873
```

Explore the CSV outputs using [Tableau](https://www.tableau.com/academic/students) or online with [RAW](http://rawgraphs.io/). For a deeper data analysis, try [Python](https://www.python.org/) or [R](https://www.datacamp.com/courses/free-introduction-to-r). The SQLite database can also be queried with [SQL](https://robots.thoughtbot.com/back-to-basics-sql). Samples of the outputted datasets can be found in the [`sample`](sample/) folder.
