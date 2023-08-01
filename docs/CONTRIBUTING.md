### Development

To work on the source code, start the app enabling debug mode with the `-d` flag and redirect stderr to a log file:
```bash
:~$ go run cmd/newscanoe/main.go -d  2> log 
```

To debug with Visual Studio Code, set the following `console` field in the `launch.json` file as follows:
```
{
    "version": "0.2.0",
    "configurations": [
        {
            [...]
            "console": "integratedTerminal",
            [...]
        }
    ]
}
```
