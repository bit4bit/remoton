# Remoton - Server

Server for remoton client/support.

By default listen at ports 9934  and 9933.

```example```
~~~
 $docker run -d --net=voipnet --ip=172.18.0.55 -e REMOTON_SERVER_AUTH_TOKEN="private" -v /opt/remoton-certs:/remoton-certs remoton-server
~~~
