Description
-------------

The consul-template's dedup manager stores the template inside consul as a key/value pair. The key is the template's hash while the value (encoded and compressed) stores the ***[]dependency.HealthService** struct. 

Given the name of the template file, this tool will create the corresponding hash (so you can use this to navigate to your consul UI), and will pull the services and corresponding nodes of these services for you.

Usage
-------------

```
./consul-template_inspector -consul localhost:8500 -file test.ctmpl
```
