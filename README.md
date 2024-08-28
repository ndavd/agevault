# agevault

`agevault` is a directory encryption tool using [age](https://github.com/FiloSottile/age) file encryption.

It locks/unlocks a vault (directory) with a passphrase-protected identity file.

Like age, it features no config options, allowing for a straightforward secure flow.

## Disclaimer

**Use it at your own risk!** (see [LICENSE](https://github.com/ndavd/agevault/blob/main/LICENSE))

Also, this is a project in early-development and hasn't been thoroughly tested. So far, I've tested it on Linux.

## Usage

```
lock/unlock directory with passphrase-protected identity file
usage: agevault [vault-name] lock|unlock|keygen
```

Securing `my-vault/`:
1. Generate identity file
```
$ agevault my-vault keygen
create identity passphrase: 
confirm identity passphrase: 
.age14tpkpl6vexufah8eq5dgrd5zy4xqs4slynh26j5n7gvxs87xhguqwu9zqc.my-vault.key.age CREATED (do not change the filename)
```
2. Lock vault
```
$ agevault my-vault lock
my-vault SECURED with age14tpkpl6vexufah8eq5dgrd5zy4xqs4slynh26j5n7gvxs87xhguqwu9zqc
```
3. Unlock vault
```
$ agevault my-vault unlock
enter passphrase for identity file ".age14tpkpl6vexufah8eq5dgrd5zy4xqs4slynh26j5n7gvxs87xhguqwu9zqc.my-vault.key.age": 
my-vault DECRYPTED
```
4. That's it. Do your changes, lock it again, etc.
