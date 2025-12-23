KEYS_DIR := keys
KEY_ACCESS_PRV := $(KEYS_DIR)/jwt-access-private.pem
KEY_ACCESS_PUB := $(KEYS_DIR)/jwt-access-public.pem
KEY_REFRESH := $(KEYS_DIR)/jwt-refresh.key

$(KEY_ACCESS_PRV) $(KEY_ACCESS_PUB) $(KEY_REFRESH):
	@mkdir -p $(KEYS_DIR)
	@openssl ecparam -genkey -name prime256v1 -noout -out $(KEY_ACCESS_PRV)
	@openssl ec -in $(KEY_ACCESS_PRV) -pubout -out $(KEY_ACCESS_PUB)
	@openssl rand -out $(KEY_REFRESH) 32

.PHONY: keys
keys: $(KEY_ACCESS_PRV) $(KEY_ACCESS_PUB) $(KEY_REFRESH)
	@echo "JWT keys generated."