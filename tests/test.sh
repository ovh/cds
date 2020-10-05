#!/bin/bash

# script usage definition
usage() { 
    echo "Usage: ./test.sh <target...>" 
    echo "   Available targets: smoke_api, smoke_services, initialization, cli, workflow, workflow_with_integration, workflow_with_third_parties"
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
VENOM_OPTS="${VENOM_OPTS:---log debug --output-dir ./results --strict --stop-on-failure}"

CDS_API_URL="${CDS_API_URL:-http://localhost:8081}"
CDS_UI_URL="${CDS_UI_URL:-http://localhost:8080}"
CDS_HATCHERY_URL="${CDS_HATCHERY_URL:-http://localhost:8086}"
CDS_HOOKS_URL="${CDS_HOOKS_URL:-http://localhost:8083}"
CDSCTL="${CDSCTL:-`which cdsctl`}"
CDSCTL_CONFIG="${CDSCTL_CONFIG:-.cdsrc}"
CDS_ENGINE_CTL="${CDS_ENGINE_CTL:-`which cds-engine`}"
SMTP_MOCK_URL="${SMTP_MOCK_URL:-http://localhost:2024}"
INIT_TOKEN="${INIT_TOKEN:-}"

# If you want to run some tests with a specific model requirements, set CDS_MODEL_REQ
CDS_MODEL_REQ="${CDS_MODEL_REQ:-buildpack-deps}"
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

smoke_tests_api() {
    echo "Running smoke tests api:"
    for f in $(ls -1 00_*.yml); do
        CMD="${VENOM} run ${VENOM_OPTS} ${f} --var cdsctl=${CDSCTL} --var api.url=${CDS_API_URL}"
        echo -e "  ${YELLOW}${f} ${DARKGRAY}[${CMD}]${NOCOLOR}"
        ${CMD} >${f}.output 2>&1
        check_failure $? ${f}.output
    done
}

initialization_tests() {
    echo "Running initialization tests:"
    CMD="${VENOM} run ${VENOM_OPTS} 01_signup.yml --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}_admin --var api.url=${CDS_API_URL} --var username=cds.integration.tests.rw --var email=it-user-rw@localhost.local --var fullname=IT_User_RW --var smtpmock.url=${SMTP_MOCK_URL} --var ring=ADMIN"
    echo -e "  ${YELLOW}01_signup.yml (admin) ${DARKGRAY}[${CMD}]${NOCOLOR}"
    ${CMD} >01_signup_admin.yml.output 2>&1
    check_failure $? 01_signup_admin.yml.output

    CMD="${VENOM} run ${VENOM_OPTS} 01_signup.yml --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}_user --var api.url=${CDS_API_URL} --var username=cds.integration.tests.ro --var email=it-user-ro@localhost.local --var fullname=IT_User_RO --var smtpmock.url=${SMTP_MOCK_URL} --var ring=USER"
    echo -e "  ${YELLOW}01_signup.yml (user) ${DARKGRAY}[${CMD}]${NOCOLOR}"
    ${CMD} >01_signup_user.yml.output 2>&1

    check_failure $? 01_signup_user.yml.output

    CMD="${VENOM} run ${VENOM_OPTS} 01_queue_stopall.yml --var cdsctl.config=${CDSCTL_CONFIG}_admin --var cdsctl=${CDSCTL} --var api.url=${CDS_API_URL}"
    echo -e "  ${YELLOW}01_queue_stopall.yml ${DARKGRAY}[${CMD}]${NOCOLOR}"
    ${CMD} >01_queue_stopall.yml.output 2>&1
    check_failure $? 01_queue_stopall.yml.output
}

smoke_tests_services() {
    echo "Running smoke tests services:"
    for f in $(ls -1 02_*.yml); do
        CMD="${VENOM} run ${VENOM_OPTS} ${f} --var cdsctl=${CDSCTL}--var ui.url=${CDS_UI_URL} --var hatchery.url=${CDS_HATCHERY_URL} --var hooks.url=${CDS_HOOKS_URL}"
        echo -e "  ${YELLOW}${f} ${DARKGRAY}[${CMD}]${NOCOLOR}"
        ${CMD} >${f}.output 2>&1
        check_failure $? ${f}.output
    done
}

cli_tests() {
    echo "Running CLI tests:"
    for f in $(ls -1 03_cli*.yml); do
        CMD="${VENOM} run ${VENOM_OPTS} ${f} --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}_admin --var engine.ctl=${CDS_ENGINE_CTL} --var api.url=${CDS_API_URL} --var ui.url=${CDS_UI_URL}  --var smtpmock.url=${SMTP_MOCK_URL}"
        echo -e "  ${YELLOW}${f} ${DARKGRAY}[${CMD}]${NOCOLOR}"
        ${CMD} >${f}.output 2>&1
        check_failure $? ${f}.output
    done
}

workflow_tests() {
    echo "Running Workflow tests:"
    for f in $(ls -1 04_*.yml); do
        CMD="${VENOM} run ${VENOM_OPTS} ${f} --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}_admin --var api.url=${CDS_API_URL} --var ui.url=${CDS_UI_URL} --var smtpmock.url=${SMTP_MOCK_URL} --var ro_username=cds.integration.tests.ro --var cdsctl.config_ro_user=${CDSCTL_CONFIG}_user"
        echo -e "  ${YELLOW}${f} ${DARKGRAY}[${CMD}]${NOCOLOR}"
        ${CMD} >${f}.output 2>&1
        check_failure $? ${f}.output
    done
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
        ${CMD} >${f}.output 2>&1
        check_failure $? ${f}.output
    done
}

workflow_with_third_parties() {
    echo "Stopping all jobs in queue:"
    CMD="${VENOM} run ${VENOM_OPTS} 01_queue_stopall.yml --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}"
    echo -e "  ${YELLOW}01_queue_stopall.yml ${DARKGRAY}[${CMD}]${NOCOLOR}"
    ${CMD} >01_queue_stopall.yml.output 2>&1
    check_failure $? 01_queue_stopall.yml.output

    if [ -z "$CDS_MODEL_REQ" ]; then echo "missing CDS_MODEL_REQ variable"; exit 1; fi
    if [ -z "$CDS_REGION_REQ" ]; then echo "missing CDS_REGION_REQ variable"; exit 1; fi
    echo "Running Workflow with third parties:"
    for f in $(ls -1 06_*.yml); do
        CMD="${VENOM} run ${VENOM_OPTS} ${f} --var cdsctl=${CDSCTL} --var cdsctl.config=${CDSCTL_CONFIG}"
        echo -e "  ${YELLOW}${f} ${DARKGRAY}[${CMD}]${NOCOLOR}"
        ${CMD} >${f}.output 2>&1
        check_failure $? ${f}.output
    done
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
            export CDS_MODEL_REQ
            export CDS_REGION_REQ
            workflow_with_third_parties;;
        *) echo -e "${RED}Error: unknown target: $target${NOCOLOR}"
            usage
            exit 1;;
    esac    
done
