KEY_REFRESH := deploy/lexigo/auth/keys/jwt-refresh.key
KEY_ACCESS_PRV := deploy/lexigo/auth/keys/jwt-access-private.pem
KEY_ACCESS_PUB := deploy/lexigo/common/keys/jwt-access-public.pem

$(KEY_ACCESS_PRV):
	@mkdir -p "$(dir $(KEY_ACCESS_PRV))"
	@openssl ecparam -genkey -name prime256v1 -noout -out $(KEY_ACCESS_PRV)

$(KEY_ACCESS_PUB): $(KEY_ACCESS_PRV)
	@mkdir -p "$(dir $(KEY_ACCESS_PUB))"
	@openssl ec -in $(KEY_ACCESS_PRV) -pubout -out $(KEY_ACCESS_PUB)

$(KEY_REFRESH):
	@mkdir -p "$(dir $(KEY_REFRESH))"
	@openssl rand -out $(KEY_REFRESH) 32

.PHONY: auth-keys
auth-keys: $(KEY_ACCESS_PRV) $(KEY_ACCESS_PUB) $(KEY_REFRESH)
	@echo "JWT keys generated."