# Vigilant Uptime Outpost

This application acts as an outpost for running uptime checks in [Vigilant](https://govigilant.io).  
It is designed to be deployed anywhere in the world to monitor services from different geographic locations.

To set up, it only requires Docker with Compose installed, along with the URL and outpost secret of your Vigilant instance.

## Architecture

When the outpost starts, it will register itself with the Vigilant instance. It will select a random port number for Vigilant to connect to.  
The outpost will then start listening for incoming requests from Vigilant. When a request is received, it will perform the uptime check and return the result back to Vigilant.  
These outposts are designed to be short-lived and can be destroyed and recreated at any time. When an outpost is destroyed, it will automatically unregister itself from Vigilant.

### Security

All communication between the outpost and Vigilant is done over HTTPS. Vigilant maintains a root CA certificate that is used to sign the outpost certificates.  
When the outpost registers itself with Vigilant, it will receive a signed certificate that it will use for all future communication.

## Deployment

Please refer to Vigilant's documentation for detailed deployment instructions: [Vigilant Deployment Guide](https://govigilant.io/documentation/deployment).

## License

This repository is licensed under the MIT License. See the [LICENSE](LICENSE.md) file for details.
