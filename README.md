# auroraops

The `auroraops` program allows you to interact with a nanoleaf aurora.

This application uses local and remote configuration to programatically update individual or groups of panels (triangles).

Demo: https://youtu.be/ePlpQkaogdI

# Configuration

There are two different configuration sets used. The first is **local** configuration that describes the behavior of statuses and how panels are grouped together by "thing". The second is **remote** configuration that tells the application what the current state of things is.

## Local Configuration

Local configuration can be described through configuration files or environment variables thanks to [viper](https://github.com/spf13/viper). The search paths for configuration are the current directory, $HOME/.auroraops/, and /etc/auroraops/. In those directories, the application will attempt to load either `auroraops.json` or `auroraops.yaml`.

A configuration file can also be set through the `server --config PATH` command line parameter.

The default location for remote configuration is `http://localhost:8080/` and the interval is 3 seconds.

There is no default configuration for either the address of the Aurora to command or the key used to authenticate. These configuration variables must be set for the application to work.

Status and thing configuration is meant to be flexible and work out of the box. A very simple configuration could have 3 status for "up" (green), "down" (red), and "unknown" (silver) and one thing called "website" that all of our panels will reflect the status of. Colors must be provided in HEX.

```
panel:
  url: "http://192.168.1.150:16021"
  key: "myspecialkey"
status:
  location: "https://ci-info.ourgreatapp.io/status.json"
  interval: 60
status:
  unknown:
    type: solid
    color: "#808080"
  up:
    type: solid
    color: "#008000"
  down:
    type: solid
    color: "#FF0000"
things:
  "website":
    panels: [13, 71, 89, 91, 250, 102, 235, 167, 11, 39, 34, 28]
```

Additionaly, the application can have `onstart` and `onstop` configuration used to "clear out" the aurora before and after use. Thins can also have individual `onstart` configuration for more complex configurations.


## Remote Configuration

Remote configuration is requested periodically through an HTTP GET request. Content type is ignored, but the body must be valid json. The parsed JSON object is just simple string key/value pairs.

```
{
  "website": "up
}
```

# Setup

To use this application, you must have authentication for the Aurora as well as know the layout of panels.

## Authentication

The `init` subcommand can be used to get authentication configuration and bootstrap a configuration file. Running the command will look for Aurora installations on your local network that are in pairing mode once. This is the process:

1. Run the `init` subcommand and wait for the prompt. (`auroraops init auroraops.yaml`)
2. Physically go the aurora and hold the power button down for 5 seconds so that you start to see a blinking light.
3. Hit "enter" and get confirmation that the configuration was written.

## Info

The `info` subcommand can be used to list all of the panels and set unique colors for each to make configuration easier. This command requires authentication via a configuration file. The `init` command be run and the configuration file include both the panel url and key.

Example output will look like:

```
Setting panel 13 to blue
Setting panel 71 to purple
Setting panel 89 to olive
Setting panel 91 to white
Setting panel 250 to grey
Setting panel 102 to red
Setting panel 235 to aqua
Setting panel 167 to teal
Setting panel 11 to fuchsia
Setting panel 39 to maroon
Setting panel 34 to yellow
Setting panel 28 to lime
```

[ ![Alt text](https://github.com/ngerakines/auroraops/raw/master/IMG_3687_tn.jpg?raw=true) ](https://github.com/ngerakines/auroraops/raw/master/IMG_3687.jpg?raw=true)

# License

Copyright 2018 Nick Gerakines

This project is open source under the MIT license.
