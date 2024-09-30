# RoPE

## To Do
- Modify client.go, server.go and proxy.go to load application logic from functions defined outside the package.
    - Check this (https://stackoverflow.com/questions/12655464/can-functions-be-passed-as-parameters) out for understandig how to use closures in golang
- **client.go**
    - Change the client so that for each connection it receves a vector of packets before closing it
    - Applications:
        - Equalize Latency
        - Fixed destination (done)
        - Fixed destination (almost done)
- **server.go**
    - Change the loadParam function to *also* read the configuration
    - Change the server so that each worker receves a vector of packets from the application and sends it back to the client
    - Applications:
        - Reply (done)
        - Reply&Forward
- **routing.go**
    - Change the client so that for each connection it receves a vector of packets before closing it
    - Change the loadParam function to *also* read the configuration
    - Applications:
        - Probabilistic (done)
        - Probabilistic with latency (done)
## Scripts
- Check launch_remote.sh
- Check push_all, push docker images to remote hosts
- Check push_docker to push docker images to dockerhub