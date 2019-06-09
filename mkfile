encrypt:V: app.yaml
	cat app.yaml | openssl aes-256-cbc >app.yaml.enc

decrypt:V:
	cat app.yaml.enc | openssl aes-256-cbc -d >app.yaml
