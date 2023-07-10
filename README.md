# webrtc-remote-handler 

This package defines code that enables cloud/web communications. 

Sample command to run:

```
go run src/* -f 'rtsp://<username>:<password>@<rtspserver>:<port>/live/ch1' -s "tcp://$MQTT_SERVER:1883" -i $(cat /etc/machine-id)
```
