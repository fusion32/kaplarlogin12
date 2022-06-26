# Kaplar Login Server

## About
This is a simple login server targetting Tibia v12. Unlike previous versions where logins were handled using a custom protocol, this version sends json encoded requests to a web server and reads back json encoded responses.

The couple scripts I found that handle this new login service were written in PHP and since it does involve setting up an apache or nginx web stack I decided to do this alternative version written in GO which is very straight forward to setup, build, and run.

## TODO
* Add HTTPS support! Without this feature, the server cannot be safely used in production.
* Implement the "eventschedule" request.
