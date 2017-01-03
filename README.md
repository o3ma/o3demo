# o3demo

This is some simple demo code, doing the following:
* Create a new identity if none exists and uses an existing one otherwise. 
* Load an address book if one exists
* Query the directory server for an ID (one of a demo echo server we run)
* Save the address book
* Start an echo server

This code needs the certificate file 'cert.pem' available next to the generated binary. It is used to validate the self-signed TLS certificate presented by the Threema directory server.
