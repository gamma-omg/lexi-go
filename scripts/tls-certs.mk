CERT_FILE := deploy/lexigo/common/tls/cert.crt
KEY_FILE := deploy/lexigo/common/tls/key.pem

$(CERT_FILE) $(KEY_FILE):
	@mkdir -p "$(dir $(CERT_FILE))"
	@mkdir -p "$(dir $(KEY_FILE))"
	@mkcert -install
	@mkcert -cert-file $(CERT_FILE) -key-file $(KEY_FILE) localhost 127.0.0.1 ::1

.PHONY: tls-certs
tls-certs: $(CERT_FILE) $(KEY_FILE)
	@echo "TLS certificates generated."