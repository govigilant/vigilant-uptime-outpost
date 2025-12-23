# Vigilant Uptime Outpost

This application acts as an outpost for running uptime checks in [Vigilant](https://govigilant.io).  
It is designed to be deployed anywhere in the world to monitor services from different geographic locations.

To set up, it only requires Docker with Compose installed, along with the URL and outpost secret of your Vigilant instance.

## Architecture

When the outpost starts, it will register itself with the Vigilant instance. It will select a random port number for Vigilant to connect to.  
The outpost will then start listening for incoming requests from Vigilant. When a request is received, it will perform the uptime check and return the result back to Vigilant.  
These outposts are designed to be short-lived and can be destroyed and recreated at any time. When an outpost is destroyed, it will automatically unregister itself from Vigilant.

### Auto-Restart

The outpost includes an auto-restart mechanism for resource efficiency. If no check requests are received for a configurable period (default: 60 minutes), the outpost will automatically shut down. Docker's restart policy (configured as `unless-stopped` in docker-compose.yml) will then start a new container, ensuring the service remains available when Vigilant does not send requests anymore.

You can configure the inactivity timeout by setting the `INACTIVITY_TIMEOUT_MINS` environment variable in your `.env` file.

### Security

All communication between the outpost and Vigilant is done over HTTPS. Vigilant maintains a root CA certificate that is used to sign the outpost certificates.  
When the outpost registers itself with Vigilant, it will receive a signed certificate that it will use for all future communication.

## Configuration

The outpost is configured using environment variables. Create a `.env` file in the root directory with the following variables:

- `VIGILANT_URL` (required): The URL of your Vigilant instance
- `OUTPOST_SECRET` (required): The secret key used to authenticate with Vigilant
- `INACTIVITY_TIMEOUT_MINS` (optional): Number of minutes of inactivity before auto-restart (default: 60)
- `PORT` (optional): The port the outpost will listen on (default: randomly assigned between 1000-10000)
- `IP` (optional): The public IP address of the outpost (default: auto-detected)
- `COUNTRY` (optional): The country associated with this outpost
- `LATITUDE` (optional): The latitude coordinate for this outpost
- `LONGITUDE` (optional): The longitude coordinate for this outpost

See `.env.example` for a sample configuration file.

## Deployment

Please refer to Vigilant's documentation for detailed deployment instructions: [Vigilant Deployment Guide](https://govigilant.io/documentation/deployment).

## License

This repository is licensed under the MIT License. See the [LICENSE](LICENSE.md) file for details.
