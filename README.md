# agevault

`agevault` is a directory encryption tool using [`age`](https://github.com/FiloSottile/age) file
encryption.

It locks/unlocks a vault (directory) with a passphrase-protected identity file.

Like `age`, it features no config options, allowing for a straightforward secure flow.

## Disclaimer

This project has been tested, but has not undergone a formal security audit, and none is planned.

The codebase is intentionally kept simple and readable, so you're encouraged to review it yourself
before relying on it for anything sensitive.

**Use at your own risk** (see [`LICENSE`](./LICENSE)).

## Installation

Always install the latest release to make sure you have the latest security improvements and fixes.
If the update has the same major version (e.g. `v1.x.x`), then it's guaranteed to be backwards
compatible.

### Download the pre-built binaries

Get them from the [latest release](https://github.com/ndavd/agevault/releases/latest).

### Using `nix`

```text
nix run github:ndavd/agevault
```

### Using `go`

```text
go install github.com/ndavd/agevault@latest
```

### Using `docker`

```text
docker build -t agevault .
docker run --rm -it -u ${UID}:${GID} -v .:/tmp agevault
```

## Usage

```text
lock/unlock directory with passphrase-protected identity file
usage: agevault [directory-name] lock|unlock|keygen
```

Assuming we want to secure the directory `my-vault/`:

1. Generate identity file

   ```text
   $ agevault my-vault keygen
   create identity passphrase: 
   confirm identity passphrase: 
   .ab2a0883204b8bb5a463fe79ecde7674.my-vault.key.age CREATED
   ab2a0883204b8bb5a463fe79ecde7674 ADDED to my-vault/.agevault-recipients
   ```

2. Lock vault

   ```text
   $ agevault my-vault lock
   my-vault LOCKED with:
     ab2a0883204b8bb5a463fe79ecde7674
   ```

3. Unlock vault

   ```text
   $ agevault my-vault unlock
   enter passphrase for identity file ".ab2a0883204b8bb5a463fe79ecde7674.my-vault.key.age": 
   my-vault UNLOCKED
   ```

4. That's it. Do your changes, lock it again, etc.

### Recipients

The vault is encrypted to every recipient listed in `<vault>/.agevault-recipients`. `keygen` adds
its recipient automatically, and the file is locked together with the rest of the vault. For
`my-vault`, it looks like this:

```text
# agevault recipients for my-vault
# lines starting with # are ignored

# fingerprint: ab2a0883204b8bb5a463fe79ecde7674
age14p8nysmdqmlhgczshculs35kt2y8rrsx9jd9mc0rglg90rr504xsh3qzty
```

To share a vault, the other person runs `agevault <vault> keygen` and sends you the recipient and
fingerprint from their `<vault>/.agevault-recipients`. Paste the recipient into yours and lock, then
send them `<vault>.age`. They need to remove the empty `<vault>/` created by their `keygen` before
unlocking.

Every recipient's fingerprint is printed on lock. Check that each one is yours or matches a
fingerprint someone sent you.

### Upgrading from versions before v1.2.0

Previously, the recipient was stored in the identity filename. Vaults created with older versions
are migrated automatically: the first `lock` creates `.agevault-recipients` from the existing
identity file. Once it's migrated, the identity file can be renamed, as long as it keeps the
`.<vault>.key.age` ending.

## Design

`agevault` relies entirely on `age` for the cryptography involved, inheriting its security. It
provides the minimal infrastructure required for `age` to work as a directory encryption tool.

`agevault` uses a passphrase-encrypted identity for a few key reasons:

- Extra security factor: Having the encrypted identity file or the passphrase by itself is useless.
- Easier to lock: If only symmetric encryption was used, the user would need the passphrase to lock
  as well; this way, the vault can be locked without entering the passphrase.
- Multiple keys ([recipients](#recipients)): Each passphrase corresponds to its own key, so multiple
  people can _own_ the vault: any one of them can lock it, and any one of them can unlock it with
  their own passphrase.

## Planned features

- [ ] Post-quantum keys
- [x] Multi-user vault support
