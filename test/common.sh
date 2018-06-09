# To be sourced from individual tests.

if [ "$V" == "1" ]; then
	set -v
fi

export TBASE="$(realpath `dirname ${0}`)"
cd ${TBASE}


# Set traps to kill our subprocesses when we exit (for any reason).
trap ":" TERM      # Avoid the EXIT handler from killing bash.
trap "exit 2" INT  # Ctrl-C, make sure we fail in that case.
trap "kill 0" EXIT # Kill children on exit.

# Generate some random test content.
# Do this for each test to avoid accidental passes.
dd if=/dev/urandom of=.random bs=1k count=20 status=none


# Wait until there's something listening on the given port.
function wait_until_ready() {
	PORT=$1

	while ! bash -c "true < /dev/tcp/localhost/$PORT" 2>/dev/null ; do
		sleep 0.1
	done
}
