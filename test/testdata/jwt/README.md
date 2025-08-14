## JWT

I used the step CLI to generate the private key and public key: https://github.com/smallstep/cli

### Generate a token

```
echo '{
  "field1": "value1",
  "nested": {
    "key": "value"
  },
  "list": ["apple", "banana"]
}' | step-cli crypto jwt sign --key priv-key --iss agentgateway.dev --aud test.agentgateway.dev --sub test-user --exp 1900650294 > example1.key
echo '{}' | step-cli crypto jwt sign --key priv-key --iss agentgateway.dev --aud test.agentgateway.dev --sub test-user --exp 1900650294 > example2.key

```

An example token is included in the directory.
