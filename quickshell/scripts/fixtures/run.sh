#!/usr/bin/env bash
# Runs the QML fixtures against the repo tree: every quickshell/ entry is
# symlinked into a scratch shell root whose shell.qml is the fixture itself.
set -euo pipefail

fixtures_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
shell_dir="$(cd -- "${fixtures_dir}/../.." && pwd)"
qs_bin="${QS:-qs}"
status=0

for fixture in "${fixtures_dir}"/*.qml; do
    name="$(basename "${fixture}" .qml)"
    harness="$(mktemp -d)"

    for entry in "${shell_dir}"/*; do
        case "$(basename "${entry}")" in
            shell.qml | scripts) continue ;;
        esac
        ln -s "${entry}" "${harness}/$(basename "${entry}")"
    done
    cp "${fixture}" "${harness}/shell.qml"

    output="$(DFILES_SOCKET="" timeout 60 "${qs_bin}" -p "${harness}" 2>&1 || true)"
    rm -rf "${harness}"

    echo "${output}" | sed -n 's/.*\[fixture\][^)]*) //p'
    if ! grep -q "FIXTURE PASS" <<<"${output}"; then
        echo "fixture ${name}: FAIL"
        status=1
        continue
    fi
    echo "fixture ${name}: PASS"
done

exit "${status}"
