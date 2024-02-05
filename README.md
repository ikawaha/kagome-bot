# kagome-bot

kagome-bot is a slack bot which tokenizes a sentence and uploads it's lattice image.

![slack_-_kuromoji](https://user-images.githubusercontent.com/4232165/32312309-8e57949c-bfe0-11e7-9664-54015c9e3be4.png)


:warning: To draw a lattice, kagome-bot uses graphviz . You need graphviz installed.


# Usage

```
$ go install github.com/ikawaha/kagome-bot
$ kagome-bot <app-level-token> <bot-token>
```

The required tokens and permissions are:

* app-level token (enable socket mode)
    * subscribe to bot events
        * message.channels
        * message.im - If you want to use in DM
* bot token
    * channels:history
    * channels:join
    * chat:write
    * files:write
    * users:read    

## slash command

The slash command supports multiple dictionaries, e.g. `/tokenize-uni`, `/tokenize-neologd`.
The bot switches dictionaries from suffix string.
The bot supports the following suffix strings.

* `uni` - [github.com/ikawaha/kagome-dict/uni](https://github.com/ikawaha/kagome-dict/uni)
* `neologd` - [github.com/ikawaha/kagome-dict-ipa-neologd](https://github.com/ikawaha/kagome-dict-ipa-neologd)
* `ipa` (default) - [github.com/ikawaha/kagome-dict/tree/master/ipa](https://github.com/ikawaha/kagome-dict/tree/master/ipa) 



# License

MIT
