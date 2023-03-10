# go-crypto-hft

A template project showing the creating of rest endpoints using Fiber, 
Mongo integration, and the installation of a custom codec to translate between Mongo's BigDecimal 
serialization format, and the 3rd party big decimal implementation from Shopspring.  This allows 
bigdecimals to be leveraged in the application, and stored transparently by mongo.
