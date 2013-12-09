# Information Retrieval Engine in Go

Learning project to teach myself Go. Implements an information
retrieval engine that supports indexing with four different types
of indexes and querying with three different ranking engines.

### To build

    cd scanner
    go build
    go install

All functionality is run through a single binary with
subcommands. Help for the configuration options is available
through a command line flag:

    scanner -h

### Indexing

The indexer supports building a number of different index types
our of the box. These are a standard single-term index, a
single-term index with positional posting list, a Porter-stemmed single-term
index, and a phrase index which considers ngrams as tokens.

The indexer also supports:

- index pruning to reduce the size of the posting lists created.
  See `-index.pruning`
- in-memory limitations which swap posting lists to disk in an
  attempt to reduce active memory usage
  See `-memlimit`

To run the indexer:

    scanner index <args>


### Querying

Querying has two components. The query engine, which reads
indices and prepares to answer queries, and the querier, which
actually makes queries. The query engine should be started first
so that it is availbe when the querier is run; it is alos

The connection between the querier and the query engine is
implemented with ZeroMQ, because I felt like learning it. Each
query can specify the preference for which index should answer
the query - if the query is not answered sufficiently by an
index, the next preferred query index be used.

To run the query engine:

    scanner start-query-engine

To run a query:

    scanner query
