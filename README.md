# Web Proxy

A very crude web proxy with support for interception made for a coursework in my second year of university.

# To-Do
Now that this coursework has been submitted I would like to add
- [ ] WebSocket Support
- [ ] Tools for encoding swapping
- [ ] Repeaters
- [ ] Proper Regex Support
- [ ] Better UI/Desktop Interface
- [ ] Support for extensions

## Usage
- To run the proxy, execute the following command:
```bash
go run .
```
- Then configure your browser to use the proxy at `localhost:8888`
- Add the cert from `localhost:8000/ca.crt` to your browser
- Open the ui at `localhost:8000`.

