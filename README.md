# kagome-bot

kagome-bot is a slack bot which tokenizes a sentence and uploads it's lattice image.

![slack_-_kuromoji](https://user-images.githubusercontent.com/4232165/32312309-8e57949c-bfe0-11e7-9664-54015c9e3be4.png)


:warning: To draw a lattice, kagome-bot uses graphviz . You need graphviz installed.


# Usage

```
$ go install github.com/ikawaha/kagome-bot
$ kagome-bot <app-level-token> <bot-token> <bot-name>
```

The required tokens and permissions are:

* app-level token (enable socket mode)
    * subscribe to bot events
        * message.channels
* bot token
    * channels:history
    * channels:join
    * chat:write
    * files:write

# License

MIT
