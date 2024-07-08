#!/bin/bash

trap 'kill $(jobs -p)' EXIT

# script usage definition
usage() {
    echo "Usage: ./test.sh <target...>"
    echo "   Available targets: smoke_api, smoke_services, initialization, cli, workflow, workflow_with_integration, workflow_with_third_parties, admin"
}

# Arguments are mandatory
[[ $# -lt 1 ]] && usage && exit 1

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
NOCOLOR='\033[0m'
RED='\033[0;31m'
GREEN='\033[0;32m'
ORANGE='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
LIGHTGRAY='\033[0;37m'
DARKGRAY='\033[1;30m'
LIGHTRED='\033[1;31m'
LIGHTGREEN='\033[1;32m'
YELLOW='\033[1;33m'
LIGHTBLUE='\033[1;34m'
LIGHTPURPLE='\033[1;35m'
LIGHTCYAN='\033[1;36m'
WHITE='\033[1;37m'

VENOM="${VENOM:-`which venom`}"
VENOM_OPTS="${VENOM_OPTS:--vv --format xml --output-dir ./results --stop-on-failure}"

CDS_API_URL="${CDS_API_URL:-http://localhost:8081}"
CDS_UI_URL="${CDS_UI_URL:-http://localhost:8080}"
CDS_HATCHERY_URL="${CDS_HATCHERY_URL:-http://localhost:8086}"
CDS_HOOKS_URL="${CDS_HOOKS_URL:-http://localhost:8083}"
CDSCTL="${CDSCTL:-`which cdsctl`}"
CDSCTL_CONFIG="${CDSCTL_CONFIG:-.cdsrc}"
CDS_ENGINE_CTL="${CDS_ENGINE_CTL:-`which cds-engine`}"
CDS_ENGINE_CONFIG="${CDS_ENGINE_CONFIG:-cds-engine.toml}"
CDS_HATCHERY_NAME="${CDS_HATCHERY_NAME:-hatchery-swarm}"
CDS_REGION="${CDS_REGION:-default}"
SMTP_MOCK_URL="${SMTP_MOCK_URL:-http://localhost:2024}"
INIT_TOKEN="${INIT_TOKEN:-}"
GITEA_USER="${GITEA_USER:-gituser}"
GITEA_PASSWORD="${GITEA_PASSWORD:-gitpwd}"
GITEA_HOST="${GITEA_HOST:-http://localhost:3000}"
GITEA_CDS_HOOKS_URL="${GITEA_CDS_HOOKS_URL:-http://localhost:8083}"
GPG_KEY_ID="${GPG_KEY_ID:-`gpg --list-secret-keys | grep --only-matching --extended-regexp "[[:xdigit:]]{40}" | head -n 1`}"

PLUGINS_DIRECTORY="${PLUGINS_DIRECTORY:-dist}"

# If you want to run some tests with a specific region requirement, set CDS_REGION_REQ
CDS_REGION_REQ="${CDS_REGION_REQ:-""}"

HOSTNAME="${HOSTNAME:-localhost}"

# The default values below fit to default minio installation.
# Run "make minio_start" to start a minio docker container
AWS_DEFAULT_REGION="${AWS_DEFAULT_REGION:-us-east-1}"
S3_BUCKET="${S3_BUCKET:-cds-it}"
AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-AKIAIOSFODNN7EXAMPLE}"
AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY}"
AWS_ENDPOINT_URL="${AWS_ENDPOINT_URL:-http://$(hostname):9000}"

echo -e "Using venom using following variables:"
echo -e "  VENOM=${CYAN}${VENOM}${NOCOLOR}"
echo -e "  VENOM_OPTS=${CYAN}${VENOM_OPTS}${NOCOLOR}"
echo ""

echo -e "Running tests using following variables:"
echo -e "  CDS_API_URL=${CYAN}${CDS_API_URL}${NOCOLOR}"
echo -e "  CDS_UI_URL=${CYAN}${CDS_UI_URL}${NOCOLOR}"
echo -e "  CDS_HATCHERY_URL=${CYAN}${CDS_HATCHERY_URL}${NOCOLOR}"
echo -e "  CDSCTL=${CYAN}${CDSCTL}${NOCOLOR}"
echo -e "  CDSCTL_CONFIG=${CYAN}${CDSCTL_CONFIG}${NOCOLOR}"
echo -e "  GPG_KEY_ID=${CYAN}${GPG_KEY_ID}${NOCOLOR}"
echo ""

check_failure() {
    exit_status=$1
    if [ $exit_status -ne 0 ]; then
        echo -e "  ${LIGHTRED}FAILURE${RED}\n"
        cat $2
        echo -e ${NOCOLOR}
        exit $exit_status
    fi
}

mv_results() {
    tsuite_file=$1
    mv ./results/venom.log ./results/${tsuite_file}-venom.log
}

smoke_tests_api() {
    echo "Running smoke tests api:"
    for f in $(ls -1 00_*.yml); do
        CMD="${VENOM} run ${VENOM_OPTS} ${f} --var cdsctl=${CDSCTL} --var api.url=${CDS_API_URL}"
        echo -e "  ${YELLOW}${f} ${DARKGRAY}[${CMD}]${NOCOLOR}"
        START="$(date +%s)"
        ${CMD} >${f}.output 2>&1
        check_failure $? ${f}.output
        echo -e "  ${DARKGRAY}duration: $[ $(date +%s) - ${START} ]${NOCOLOR}"
        mv_results ${f}
    done
}

initialization_tests() {
    echo "Running initialization tests:"
    CMD="${VENOM} run ${VENOM_OPTS} 01_signup.yml --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}_admin --var api.url=${CDS_API_URL} --var username=cds.integration.tests.rw --var email=it-user-rw@localhost.local --var fullname=IT_User_RW --var smtpmock.url=${SMTP_MOCK_URL} --var ring=ADMIN"
    echo -e "  ${YELLOW}01_signup.yml (admin) ${DARKGRAY}[${CMD}]${NOCOLOR}"
    ${CMD} >01_signup_admin.yml.output 2>&1
    check_failure $? 01_signup_admin.yml.output
    mv_results ${f}

    CMD="${VENOM} run ${VENOM_OPTS} 01_signup.yml --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}_user --var api.url=${CDS_API_URL} --var username=cds.integration.tests.ro --var email=it-user-ro@localhost.local --var fullname=IT_User_RO --var smtpmock.url=${SMTP_MOCK_URL} --var ring=USER"
    echo -e "  ${YELLOW}01_signup.yml (user) ${DARKGRAY}[${CMD}]${NOCOLOR}"
    ${CMD} >01_signup_user.yml.output 2>&1
    check_failure $? 01_signup_user.yml.output
    mv_results ${f}

    CMD="${VENOM} run ${VENOM_OPTS} 01_queue_stopall.yml --var cdsctl.config=${CDSCTL_CONFIG}_admin --var cdsctl=${CDSCTL} --var api.url=${CDS_API_URL}"
    echo -e "  ${YELLOW}01_queue_stopall.yml ${DARKGRAY}[${CMD}]${NOCOLOR}"
    ${CMD} >01_queue_stopall.yml.output 2>&1
    check_failure $? 01_queue_stopall.yml.output
    mv_results ${f}

    CMD="${VENOM} run ${VENOM_OPTS} 01_init_hatchery.yml --var cdsctl.config=${CDSCTL_CONFIG}_admin --var cdsctl=${CDSCTL} --var api.url=${CDS_API_URL} --var engine.ctl=${CDS_ENGINE_CTL} --var engine.config=${CDS_ENGINE_CONFIG} --var hatchery.name=${CDS_HATCHERY_NAME}"
    echo -e "  ${YELLOW}01_init_hatchery.yml ${DARKGRAY}[${CMD}]${NOCOLOR}"
    ${CMD} >01_init_hatchery.yml.output 2>&1
    check_failure $? 01_init_hatchery.yml.output
    mv_results ${f}

    CMD="${VENOM} run ${VENOM_OPTS} 01_init_plugins.yml --var cdsctl.config=${CDSCTL_CONFIG}_admin --var cdsctl=${CDSCTL} --var api.url=${CDS_API_URL} --var engine.ctl=${CDS_ENGINE_CTL} --var dist=${PLUGINS_DIRECTORY}"
    echo -e "  ${YELLOW}01_init_plugins.yml ${DARKGRAY}[${CMD}]${NOCOLOR}"
    ${CMD} >01_init_hatchery.yml.output 2>&1
    check_failure $? 01_init_hatchery.yml.output
    mv_results ${f}
}

smoke_tests_services() {
    echo "Running smoke tests services:"
    for f in $(ls -1 02_*.yml); do
        CMD="${VENOM} run ${VENOM_OPTS} ${f} --var cdsctl.config=${CDSCTL_CONFIG}_admin --var cdsctl=${CDSCTL} --var ui.url=${CDS_UI_URL} --var hatchery.url=${CDS_HATCHERY_URL} --var hooks.url=${CDS_HOOKS_URL}"
        echo -e "  ${YELLOW}${f} ${DARKGRAY}[${CMD}]${NOCOLOR}"
        START="$(date +%s)"
        ${CMD} >${f}.output 2>&1
        check_failure $? ${f}.output
        echo -e "  ${DARKGRAY}duration: $[ $(date +%s) - ${START} ]${NOCOLOR}"
        mv_results ${f}
    done
}

cli_tests() {
    echo "Running CLI tests:"
    for f in $(ls -1 03_cli*.yml); do
        CMD="${VENOM} run ${VENOM_OPTS} ${f} --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}_admin --var engine.ctl=${CDS_ENGINE_CTL} --var api.url=${CDS_API_URL} --var ui.url=${CDS_UI_URL}  --var smtpmock.url=${SMTP_MOCK_URL}"
        echo -e "  ${YELLOW}${f} ${DARKGRAY}[${CMD}]${NOCOLOR}"
        START="$(date +%s)"
        ${CMD} >${f}.output 2>&1
        check_failure $? ${f}.output
        echo -e "  ${DARKGRAY}duration: $[ $(date +%s) - ${START} ]${NOCOLOR}"
        mv_results ${f}
    done
}

workflow_tests() {
    max_children=10
    echo "Running Workflow tests"
    for f in $(ls -1 04_*.yml); do
        run_workflow_tests $f &
        local my_pid=$$
        local children=$(ps -eo ppid | grep -w $my_pid | wc -w)
        children=$((children-1))
        if [[ $children -ge $max_children ]]; then
            wait -n
        fi
    done
    wait
}

run_workflow_tests() {
    f=$1
    rm -rf ./results/${f} && mkdir -p ./results/${f}
    CMD="${VENOM} run ${VENOM_OPTS} --output-dir ./results/${f} ${f} --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}_admin --var api.url=${CDS_API_URL} --var ui.url=${CDS_UI_URL} --var smtpmock.url=${SMTP_MOCK_URL} --var ro_username=cds.integration.tests.ro --var cdsctl.config_ro_user=${CDSCTL_CONFIG}_user"
    echo -e "  ${YELLOW}${f} ${BLUE}STARTING ${DARKGRAY}cmd: ${CMD}${NOCOLOR}"
    START="$(date +%s)"
    ${CMD} > ./results/${f}/${f}.output 2>&1
    exit_status=$?    
    if [ $exit_status -ne 0 ]; then
        out=`cat ./results/${f}/${f}.output`
        echo -e "  ${YELLOW}${f} ${LIGHTRED}FAILURE ${DARKGRAY}code: ${exit_status}\n${RED}${out}${NOCOLOR}"
        mv ./results/${f}/venom.log ./results/${f}/${f}-venom.log
    else
        echo -e "  ${YELLOW}${f} ${GREEN}SUCCESS ${DARKGRAY}duration: $[ $(date +%s) - ${START} ]${NOCOLOR}"
    fi
    mv ./results/${f}/* ./results
    exit $exit_status
}

workflow_with_integration_tests() {
    if [ -z "$OS_AUTH_URL" ]; then echo "missing OS_* variables"; exit 1; fi
    if [ -z "$OS_REGION_NAME" ]; then echo "missing OS_* variables"; exit 1; fi
    if [ -z "$OS_TENANT_NAME" ]; then echo "missing OS_* variables"; exit 1; fi
    if [ -z "$OS_USERNAME" ]; then echo "missing OS_* variables"; exit 1; fi
    if [ -z "$OS_PASSWORD" ]; then echo "missing OS_* variables"; exit 1; fi
    echo "Running Workflow with Storage integration tests:"
    for f in $(ls -1 05_*.yml); do
        CMD="${VENOM} run ${VENOM_OPTS} ${f} --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}_admin --var api.url=${CDS_API_URL} --var ui.url=${CDS_UI_URL} --var smtpmock.url=${SMTP_MOCK_URL}"
        echo -e "  ${YELLOW}${f} ${DARKGRAY}[${CMD}]${NOCOLOR}"
        START="$(date +%s)"
        ${CMD} >${f}.output 2>&1
        check_failure $? ${f}.output
        echo -e "  ${DARKGRAY}duration: $[ $(date +%s) - ${START} ]${NOCOLOR}"
        mv_results ${f}
    done
}

workflow_with_third_parties() {
    echo "Stopping all jobs in queue:"
    CMD="${VENOM} run ${VENOM_OPTS} 01_queue_stopall.yml --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}"
    echo -e "  ${YELLOW}01_queue_stopall.yml ${DARKGRAY}[${CMD}]${NOCOLOR}"
    ${CMD} >01_queue_stopall.yml.output 2>&1
    check_failure $? 01_queue_stopall.yml.output
    mv_results ${f}

    if [ -z "$CDS_REGION_REQ" ]; then echo "missing CDS_REGION_REQ variable"; exit 1; fi
    if [ -z "$VENOM_VAR_projectKey" ]; then echo "missing VENOM_VAR_projectKey variable"; exit 1; fi
    if [ -z "$VENOM_VAR_integrationName" ]; then echo "missing VENOM_VAR_integrationName variable"; exit 1; fi

    echo "Running Workflow with third parties:"
    for f in $(ls -1 06_*.yml); do
        CMD="${VENOM} run ${VENOM_OPTS} ${f} --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}"
        echo -e "  ${YELLOW}${f} ${DARKGRAY}[${CMD}]${NOCOLOR}"
        START="$(date +%s)"
        ${CMD} >${f}.output 2>&1
        check_failure $? ${f}.output
        echo -e "  ${DARKGRAY}duration: $[ $(date +%s) - ${START} ]${NOCOLOR}"
        mv_results ${f}
    done
}

admin_tests() {
    echo "Running Admin tests:"
    for f in $(ls -1 07_*.yml); do
        CMD="${VENOM} run ${VENOM_OPTS} ${f} --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}_admin --var api.url=${CDS_API_URL} --var ui.url=${CDS_UI_URL} --var smtpmock.url=${SMTP_MOCK_URL} --var ro_username=cds.integration.tests.ro --var cdsctl.config_ro_user=${CDSCTL_CONFIG}_user"
        echo -e "  ${YELLOW}${f} ${DARKGRAY}[${CMD}]${NOCOLOR}"
        START="$(date +%s)"
        ${CMD} >${f}.output 2>&1
        check_failure $? ${f}.output
        echo -e "  ${DARKGRAY}duration: $[ $(date +%s) - ${START} ]${NOCOLOR}"
        mv_results ${f}
    done
}

cds_v2_tests() {
    max_children=10
    echo "Check if gitea is running"
    curl --fail -I -X GET ${GITEA_HOST}/api/swagger
    echo "Running CDS v2 tests:"
    for f in $(ls -1 08_*.yml); do
        run_cds_v2_tests $f &
        local my_pid=$$
        local children=$(ps -eo ppid | grep -w $my_pid | wc -w)
        children=$((children-1))
        if [[ $children -ge $max_children ]]; then
            wait -n
        fi
    done
    wait
}

run_cds_v2_tests() {
    f=$1
    rm -rf ./results/${f} && mkdir -p ./results/${f}
    CMD="${VENOM} run ${VENOM_OPTS} --output-dir ./results/${f} ${f} --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}_admin --var api.url=${CDS_API_URL} --var ui.url=${CDS_UI_URL} --var smtpmock.url=${SMTP_MOCK_URL} --var ro_username=cds.integration.tests.ro --var cdsctl.config_ro_user=${CDSCTL_CONFIG}_user --var gitea.hook.url=${GITEA_CDS_HOOKS_URL} --var git.host=${GITEA_HOST} --var git.user=${GITEA_USER} --var git.password=${GITEA_PASSWORD} --var engine=${CDS_ENGINE_CTL} --var hatchery.name=${CDS_HATCHERY_NAME} --var gpg.key_id=${GPG_KEY_ID} --var cds.region=${CDS_REGION}"
    echo -e "  ${YELLOW}${f} ${BLUE}STARTING ${DARKGRAY}cmd: ${CMD}${NOCOLOR}"
    START="$(date +%s)"
    ${CMD} > ./results/${f}/${f}.output 2>&1
    exit_status=$?    
    if [ $exit_status -ne 0 ]; then
        out=`cat ./results/${f}/${f}.output`
        echo -e "  ${YELLOW}${f} ${LIGHTRED}FAILURE ${DARKGRAY}code: ${exit_status}\n${RED}${out}${NOCOLOR}"
        mv ./results/${f}/venom.log ./results/${f}/${f}-venom.log
    else
        echo -e "  ${YELLOW}${f} ${GREEN}SUCCESS ${DARKGRAY}duration: $[ $(date +%s) - ${START} ]${NOCOLOR}"
    fi
    mv ./results/${f}/* ./results
    exit $exit_status
}

rm -rf ./results
mkdir results

for target in $@; do
    case $target in
        smoke_api)
            smoke_tests_api;;
        initialization)
            initialization_tests;;
        smoke_services)
            smoke_tests_services;;
        cli)
            cli_tests;;
        workflow)
            workflow_tests;;
        workflow_with_integration)
            export AWS_DEFAULT_REGION
            export S3_BUCKET
            export AWS_ACCESS_KEY_ID
            export AWS_SECRET_ACCESS_KEY
            export AWS_ENDPOINT_URL
            workflow_with_integration_tests;;
        workflow_with_third_parties)
            export CDS_REGION_REQ
            workflow_with_third_parties;;
        admin)
            admin_tests;;
        v2)
            cds_v2_tests;;
        *) echo -e "${RED}Error: unknown target: $target${NOCOLOR}"
            usage
            exit 1;;
    esac
done
