#!/bin/sh
#
# Copyright (c) 2016 Jeromy Johnson
# MIT Licensed; see the LICENSE file in this repository.
#

test_description="Test ipfs repo fsck"

. lib/test-lib.sh

test_init_ipfs

test_expect_success "ipfs repo rm-root" '
  ipfs repo rm-root | tee rm-root.actual &&
  echo "Unlinked files API root.  Root hash was QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn." > rm-root.expected &&
  test_cmp rm-root.expected rm-root.actual
'

test_expect_success "files api root really removed" '
  ipfs repo rm-root | tee rm-root-post.actual &&
  echo "Files API root not found." > rm-root-post.expected &&
  test_cmp rm-root-post.expected rm-root-post.actual
'

test_launch_ipfs_daemon

test_expect_success "ipfs repo rm-root does not run on daemon" '
  test_must_fail ipfs repo rm-root
'

test_kill_ipfs_daemon

test_done
