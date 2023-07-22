# Tuya to MQTT

## Introduction

## MQTT

### Topic Structure

#### Template: 
```/$prefix/$company_name/$product_id/$device_id/status/$property_name```

Parameters:
- prefix: the mqtt topic set as program param
- company_name: the name of the company that produced product
- product_id: the id of the product
- device_id: the product device id
- property_name: the product property being published

#### Example 1: The property by property publish

Topic: ```/topic/smarthome/devices/tuya/dmi4usmu/vdevo167493813773312/status/va_temperature```

Parameters:
- prefix: topic/smarthome/devices
- company_name: tuya
- product_id: dmi4usmu
- device_id: vdevo167493813773312
- property_name: va_temperature

Value:
```22``` (text)

#### Example 2: The multi-property publish(toya-mqtt pass-through)

Topic: ```/topic/smarthome/devices/tuya/dmi4usmu/vdevo167493813773312/status/all```

Parameters:
- prefix: topic/smarthome/devices
- company_name: tuya
- product_id: dmi4usmu
- device_id: vdevo167493813773312
- property_name: all (special case)

Value:
```json
{
   "dataId":"89c5dcf0-9fc6-11ed-abd9-1a4883ee6e3b",
   "devId":"vdevo167493813773312",
   "productKey":"dmi4usmu",
   "status":[
      {
         "code":"va_temperature",
         "t":1674991058214,
         "value":29
      },
      {
         "code":"va_humidity",
         "t":1674991058214,
         "value":28
      },
      {
         "code":"battery_state",
         "t":1674991058214,
         "value":"middle"
      },
      {
         "code":"battery_percentage",
         "t":1674991058214,
         "value":78
      }
   ]
}
```
### Debugging

#### Subscribe

Command:
```bash
tuya_to_mqtt --tuya-user=*** --tuya-access-id=*** --tuya-access-key=*** --mqtt-url=tcp://localhost:1883 --mqtt-client-id=test --mqtt-username=*** --mqtt-password=*** --mqtt-topic=topic/test/# subscribe
```

Output:
```text
topic/test/tuya/dmi4usmu/vdevo167493813773312/status/va_temperature 1718
topic/test/tuya/dmi4usmu/vdevo167493813773312/status/va_humidity 20
topic/test/tuya/dmi4usmu/vdevo167493813773312/status/battery_state high
topic/test/tuya/dmi4usmu/vdevo167493813773312/status/battery_percentage 42
```

#### Publish

Command:
```bash
tuya_to_mqtt --tuya-user=*** --tuya-access-id=*** --tuya-access-key=*** --mqtt-url=tcp://localhost:1883 --mqtt-client-id=test2 --mqtt-username=*** --mqtt-password=*** --mqtt-topic=topic/test/tuya/dmi4usmu/vdevo167493813773312/status publish --json={"dataId":"371ac9f3-9f61-11ed-abd9-1a4883ee6e3b","devId":"vdevo167493813773312","productKey":"dmi4usmu","status":[{"code":"va_temperature","t":1674947540350,"value":1718},{"code":"va_humidity","t":1674947540350,"value":20},{"code":"battery_state","t":1674947540350,"value":"high"},{"code":"battery_percentage","t":1674947540350,"value":42}]}
```