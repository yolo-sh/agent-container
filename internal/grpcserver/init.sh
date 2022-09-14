#!/bin/bash
# Yolo environment container init script.
set -euo pipefail

log () {
  echo -e "${1}" >&2
}

# Remove "debconf: unable to initialize frontend: Dialog" warnings
echo 'debconf debconf/frontend select Noninteractive' | sudo tee debconf-set-selections > /dev/null

handleExit () {
  EXIT_CODE=$?
  exit "${EXIT_CODE}"
}

trap "handleExit" EXIT

# -- Run as "yolo"

log "Configuring workspace for user \"yolo\""

sudo --set-home --login --user yolo -- env \
	GITHUB_USER_EMAIL="${GITHUB_USER_EMAIL}" \
	USER_FULL_NAME="${USER_FULL_NAME}" \
bash << 'EOF'

if [[ ! -f ".ssh/yolo-github" ]]; then
	ssh-keygen -t ed25519 -C "${GITHUB_USER_EMAIL}" -f .ssh/yolo-github -q -N ""
fi

chmod 644 .ssh/yolo-github.pub
chmod 600 .ssh/yolo-github

if ! grep --silent --fixed-strings "IdentityFile ~/.ssh/yolo-github" .ssh/config; then
	rm --force .ssh/config
  echo "Host github.com" >> .ssh/config
	echo "  User git" >> .ssh/config
	echo "  Hostname github.com" >> .ssh/config
	echo "  PreferredAuthentications publickey" >> .ssh/config
	echo "  IdentityFile ~/.ssh/yolo-github" >> .ssh/config
fi

chmod 600 .ssh/config

if ! grep --silent --fixed-strings "github.com" .ssh/known_hosts; then
  ssh-keyscan github.com >> .ssh/known_hosts
fi

GIT_GPG_KEY_COUNT="$(gpg --list-signatures --with-colons | grep 'sig' | grep "${GITHUB_USER_EMAIL}" | wc -l)"

if [[ $GIT_GPG_KEY_COUNT -eq 0 ]]; then
	gpg --quiet --batch --gen-key << EOF2
%no-protection
Key-Type: RSA
Key-Length: 4096
Subkey-Type: RSA
Subkey-Length: 4096
Name-Real: ${USER_FULL_NAME}
Name-Email: ${GITHUB_USER_EMAIL}
Expire-Date: 0
EOF2
fi

GIT_GPG_KEY_ID="$(gpg --list-signatures --with-colons | grep 'sig' | grep "${GITHUB_USER_EMAIL}" | head --lines 1 | cut --delimiter ':' --fields 5)"

if [[ ! -f ".gnupg/yolo-github-gpg-public.pgp" ]]; then
	GIT_GPG_PUBLIC_KEY="$(gpg --armor --export "${GIT_GPG_KEY_ID}")"

	echo "${GIT_GPG_PUBLIC_KEY}" >> .gnupg/yolo-github-gpg-public.pgp
fi

chmod 644 .gnupg/yolo-github-gpg-public.pgp

if [[ ! -f ".gnupg/yolo-github-gpg-private.pgp" ]]; then
	GIT_GPG_PRIVATE_KEY="$(gpg --armor --export-secret-keys "${GIT_GPG_KEY_ID}")"

	echo "${GIT_GPG_PRIVATE_KEY}" >> .gnupg/yolo-github-gpg-private.pgp
fi

chmod 600 .gnupg/yolo-github-gpg-private.pgp

git config --global pull.rebase false

git config --global user.name "${USER_FULL_NAME}"
git config --global user.email "${GITHUB_USER_EMAIL}"

git config --global user.signingkey "${GIT_GPG_KEY_ID}"
git config --global commit.gpgsign true

EOF
