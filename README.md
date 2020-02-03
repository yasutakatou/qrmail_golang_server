# qrmail_golang_server

メールやファイルを補完するgo製のAPIサーバーです<br>

# 前提<br>

 - 各メールのデータストアとしてredisを使用するのでaptなどでredisを起動させてください<br>
 - 通信暗号化のためにSSL証明書を使うので以下ブログを参考にmkcertなどでlocalhost宛のオレオレ証明書(と認証局)を作成してください<br>
[数分でできる！mkcertでローカル環境へのSSL証明書設定](https://www.hivelocity.co.jp/blog/46149/)<br>
　ファイル名はlocalhost.pem、localhost-key.pemとしてください
 - メール部分はGMAILのAPIを使うので以下ブログを参考にcredentials.jsonを作成してください<br>
[【図多め】APIを使ってGoogleサービスを操る](https://tech-blog.rakus.co.jp/entry/20180725/google-apis/google-cloud-platform/quickstart)
 - 内部からHyperledgerの**node query.js**へのlocalhost宛て認証を行うので先に**node query.js**でHyperledgerと認証するAPIを起動する必要があります。<br>
[詳しくはこちらから](https://github.com/yasutakatou/fabcar)

# 使い方<br>

動作させる環境にあわせて以下オプションを起動時に設定します<br>

```
("api","127.0.0.1:8887","[-api=API Server and Port]")
("db","127.0.0.1","[-db=DB Server and Port]")
("redis","127.0.0.1:6379","[-redis=REDIS Server and Port]")
```

-apiはスマホが認証用の問い合わせAPIのIP:PORTを指定します。QRコードを読んだらこのIP:PORTにPOSTが送られます。<br>
-dbはgolangが動作するサーバーのIP:PORTを指定します。Web画面の各APIリクエストはこのIP:PORTに送られます。<br>
-redisはredis-serverのIP:PORTを指定します。このコードからredisへの問い合わせ先です。<br>

例えば10.0.0.1のアドレスでこのコードを動かすのであれば以下のように指定します。

```
go run srv.go -db=10.0.0.1 -api=10.0.0.1:
```

(このコードはローカルのPORT:28080でAPIリクエストを受ける。応答は10.0.0.1から返答。redisはローカルを使う。<br>
 hyperledgerの認証APIは10.0.0.1:38080で受ける。※ここをlocalにするとスマホの127.0.0.1への問い合わせになってしまうので注意！)<br>
