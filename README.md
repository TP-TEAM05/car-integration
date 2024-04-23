# Car Integration

This submodule serves as an integration between cars and other modules in the solution. Cars connect to it and send information about their parameters. When a car connects to the integration, it automatically subscribes to decision updates.

## Subscription Data

### Car Data

Received data from cars are resent to every module that subscribed to it. There are two modes to subscribe to car updates:

- **Live Updates**: Sends every update from the car as soon as it receives it.
- **Periodic Updates**: Sends the data only at periodic intervals that can be specified in the subscription packet. This mode needs a "topic" parameter – the topic can be "car" or "network-statistics".

#### Data sent from car has the following attributes:

```go
type UpdateVehicleVehicle struct {
    Id                  int     
    Vin                 string  
    IsControlledByUser  bool    
    Longitude           float32 
    Latitude            float32 
    GpsDirection        float32 
    FrontUltrasonic     float32 
    FrontLidar          float32 
    RearUltrasonic      float32 
    Speed               float32 
    SpeedFrontLeft      float32 
    SpeedFrontRight     float32 
    SpeedRearLeft       float32 
    SpeedRearRight      float32 
}
```
### Decision updates data
Decision updates from decision-module are automatically sent separately to each connected car by their VIN. The message has following structure:

```go
type UpdateVehicleDecision struct {
    Vin       	string 
    Message   	string 
}
```
-	message contains direction and speed calculated by decision module.

### Network statistics
Network statistics can be sent to subscribed submodule by specifying topic parameter as „network-statistics“. The message consists of following information:

```go
type NetworkStatistics struct {
    PacketsReceived 	int64 
    ReceiveErrors   	int64 
    AverageLatency  	int64 
    Jitter          		int64 
}
```

## Subscription Logic
The subscription logic operates by awaiting synchronization conditions, which are triggered upon the reception of a packet. The sync conditions are in a DataModel class.
