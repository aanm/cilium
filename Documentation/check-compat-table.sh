#!/usr/bin/env bash

dir=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

set -o errexit
set -o nounset
set -o pipefail

# Ensure sort order doesn't depend on locale
export LANG=C
export LC_ALL=C

get_schema_of_tag(){
   tag="${1}"
   git grep -o 'CustomResourceDefinitionSchemaVersion =.*' ${tag} -- pkg/k8s | sed 's/.*=\ "//;s/"//'
}

get_schema_of_branch(){
   stable_branch="${1}"
   git grep -o 'CustomResourceDefinitionSchemaVersion =.*' origin/${stable_branch} -- pkg/k8s | sed 's/.*=\ "//;s/"//'
}

get_stable_branches(){
   git grep -o -E 'tree\/v[^>]+' -- README.rst | sed 's+.*tree/++' | sort -n
}

get_stable_tags_for_minor(){
   minor_ver="${1}"
   git tag | grep "^${minor_ver}" | grep -v '\-' | sort -V
}

get_rc_tags_for_minor(){
   minor_ver="${1}"
   git tag | grep "^${minor_ver}" | grep '\-' | sort -V
}

dst_file="${dir}/concepts/kubernetes/compatibility-table.rst"

git fetch --tags

echo   "+-----------------+----------------+" > "${dst_file}"
echo   "| Cilium          | CNP and CCNP   |" >> "${dst_file}"
echo   "| Version         | Schema Version |" >> "${dst_file}"
echo   "+-----------------+----------------+" >> "${dst_file}"
for stable_branch in $(get_stable_branches); do
    rc_tags=$(get_rc_tags_for_minor "${stable_branch}")
    stable_tags=$(get_stable_tags_for_minor "${stable_branch}")
    for tag in ${rc_tags} ${stable_tags}; do
        schema_version=$(get_schema_of_tag "${tag}")
        printf "| %-15s | %-14s |\n" ${tag} ${schema_version} >> "${dst_file}"
        echo   "+-----------------+----------------+" >> "${dst_file}"
    done
    schema_version=$(get_schema_of_branch "${stable_branch}")
    printf "| %-15s | %-14s |\n" ${stable_branch} ${schema_version} >> "${dst_file}"
    echo   "+-----------------+----------------+" >> "${dst_file}"
done

schema_version=$(get_schema_of_branch "master")
printf "| %-15s | %-14s |\n" "latest / master" ${schema_version} >> "${dst_file}"
echo   "+-----------------+----------------+" >> "${dst_file}"

exit 0
