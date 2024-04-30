# Car Integration

This submodule serves as an integration between cars and other modules in the solution. Cars connect to it and send information about their parameters. When a car connects to the car integration module, integration automatically creates subscription to decision updates for car.

## Subscription Data

### Car Data

Received data from cars are resent to every module that subscribed to it. There are two modes to subscribe to car updates:

- **Live Updates**: Sends every update from the car as soon as it receives it.
- **Periodic Updates**: Sends the data only at periodic intervals that can be specified in the subscription 

Car integration can provide updates about cars or network updates, based on specified topic in subscription packet:
- **Car**
- **Network updates** 

### Decision updates data
Decision updates from decision-module are automatically sent separately to each connected car by their VIN. 
-	message contains direction and speed calculated by decision module.

### Network statistics
Network statistics can be sent to subscribed submodule by specifying topic parameter as „network-statistics“.

## Subscription Logic
The subscription logic operates by awaiting synchronization conditions, which are triggered upon the reception of a packet. The sync conditions are in a DataModel class.
