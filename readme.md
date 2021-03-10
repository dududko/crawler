

## Tests

* The unit-tests with prefix `TestIntegration` are only runnable if the dummy HTTP server identified in the `docker-compose.yaml` file as the **site** service is up.
* Even though I support the idea of unit-testing only the public-interface (exported-methods) of a given package as a way to allow implementation details to vary without impacting behavior from customer/client code in this repo I ended-up testing some non-exported methods as a strategy to test some more localized corner-cases without the need to put in place a server or an integration test for that purpose (e.g. the 'TestValidURLs' unit-test )


## limitations

* valid links are considered 
* only links which point to real html pages are supported, served directories are not supported

