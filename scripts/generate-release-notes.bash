#!/bin/bash

set -eu
set -o pipefail

THIS_FILE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
CI="${THIS_FILE_DIR}/../../wg-app-platform-runtime-ci"
. "$CI/shared/helpers/release-note-helpers.bash"
. "$CI/shared/helpers/git-helpers.bash"
unset THIS_FILE_DIR

# ex. version_range="v0.343.0...v0.344.0"
version_range="${1:?Please provide the start and end versions you want to generate release notes for './generate-release-notes.bash START_REF...END_REF' }"
# ex. local_start_ref="v0.343.0"
local_start_ref=$(get_start_ref_from_range "${version_range}")
# ex. local_end_ref="v0.344.0"
local_end_ref=$(get_end_ref_from_range "${version_range}")

GO_MOD_LOCATION="src/code.cloudfoundry.org/go.mod";
BLOBS_LOCATION="config/blobs.yml";

get_non_bot_commits "${local_start_ref}" "${local_end_ref}"
echo ""

START_REF_ROUTING_API=$(git rev-parse "${local_start_ref}:src/code.cloudfoundry.org/routing-api")
END_REF_ROUTING_API=$(git rev-parse "${local_end_ref}:src/code.cloudfoundry.org/routing-api")
pushd src/code.cloudfoundry.org/routing-api > /dev/null
  get_non_bot_commits "${START_REF_ROUTING_API}" "${END_REF_ROUTING_API}" "routing-api"
popd > /dev/null

echo ""
display_blob_change_info "${local_start_ref}" "${local_end_ref}" "${BLOBS_LOCATION}"
echo ""
display_go_mod_diff "${local_start_ref}" "${local_end_ref}" "${GO_MOD_LOCATION}"
echo ""
display_go_mod_diff "${local_start_ref}" "${local_end_ref}" "src/routing_utils/nats_client/go.mod" "nats-client"
