#!/usr/bin/env bash
set -e

# RED='\033[0;31m'
# LIGHT_RED='\033[1;31m'
# GREEN='\033[0;32m'
LIGHT_GREEN='\033[1;32m'
NC='\033[0m'

# ERROR_TEXT="${LIGHT_RED}ERROR:${NC}"

echo -e "${LIGHT_GREEN}=== Starting the build process ===${NC}"

GIT_INFO=$(git describe --tags)
DATE_INFO=$(date '+%Y-%m-%d_%H-%M-%S')
VERSION=${GIT_INFO}_COMPILED_ON_${DATE_INFO} #; _=${VERSION}
# Set the RELEASE=true, to build the release version
if test -z "${RELEASE}"; then
    echo -e "${LIGHT_GREEN}Building the DEV version of HosterCore${NC}"
    echo ""

    printf "Building the ${LIGHT_GREEN}hoster${NC} module ... "
    # go build -a -ldflags="-X HosterCore/cmd.HosterVersion=${VERSION}" -o hoster
    go build -ldflags="-X HosterCore/cmd.HosterVersion=${VERSION}" -o hoster
else
    echo -e "${LIGHT_GREEN}Building the RELEASE version of HosterCore${NC}"
    echo ""

    printf "Building the ${LIGHT_GREEN}hoster${NC} module ... "
    go build -o hoster
fi
printf "${LIGHT_GREEN}Done${NC}\n"

printf "Building the ${LIGHT_GREEN}vm_supervisor_service${NC} module ... "
cd vm_supervisor/
go build
printf "${LIGHT_GREEN}Done${NC}\n"

printf "Building the ${LIGHT_GREEN}self_update_service${NC} module ... "
cd ..
cd self_update/
go build
printf "${LIGHT_GREEN}Done${NC}\n"

printf "Building the ${LIGHT_GREEN}dns_server${NC} module ... "
cd ..
cd dns_server/
go build
printf "${LIGHT_GREEN}Done${NC}\n"

printf "Building the ${LIGHT_GREEN}mbuffer${NC} limiter module ... "
cd ..
cd mbuffer/
go build
printf "${LIGHT_GREEN}Done${NC}\n"

printf "Building the ${LIGHT_GREEN}node_exporter_custom${NC} module ... "
cd ..
cd node_exporter/
go build
printf "${LIGHT_GREEN}Done${NC}\n"

printf "Building the ${LIGHT_GREEN}hoster_rest_api${NC} module ... "
cd ..
cd rest_api/
go build
printf "${LIGHT_GREEN}Done${NC}\n"

printf "Building the ${LIGHT_GREEN}ha_watchdog${NC} module ... "
cd ..
cd ha_watchdog/
go build
printf "${LIGHT_GREEN}Done${NC}\n"

echo -e "${LIGHT_GREEN}=== Build process done ===${NC}"
