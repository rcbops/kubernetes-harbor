# Harbor with Rackspace Managed Kubernetes Auth

This section explains the development environment of Harbor with Rackspace Managed Kubernetes Auth on Mac OSX.

Read the [Harbor](#harbor) section. Because master is unstable, all development happens in the [rackspace-mk8s-auth branch](https://github.com/rcbops/kubernetes-harbor/tree/rackspace-mk8s-auth) which is always based on a stable release branch. The rackspace-mk8s-auth branch will need to be rebased onto newer Harbor stable branches as Harbor evolves.

The only thing in the master branch is this section of the README.

## Dev Env

There are a few things you need to do to get a local dev env on Mac OSX going.

1. Fork the repo.

1. Even though the repo is git@github.com:rcbops/kubernetes-harbor.git it's easier to clone it to a directory structure where Go lang expects everything to be.

    ```bash
    mkdir $GOPATH/src/github.com/vmware
    git clone git@github.com:my-github-username/kubernetes-harbor.git $GOPATH/src/github.com/vmware/harbor
    cd $GOPATH/src/github.com/vmware/harbor
    git remote add upstream git@github.com:rcbops/kubernetes-harbor.git
    ```

1. Create a data dir.

    ```bash
    sudo mkdir /data
    sudo chown $(whoami):staff /data
    ```

1. Open your Docker preferences and under File Sharing add Docker File `/data` and `/var/log`.

## Run Harbor

This is cribbed from the [compile guide](docs/compile_guide.md). This deploys all Harbor components using your local Docker/Docker Compose. A few changes have been made to [harbor.cfg](make/harbor.cfg) to make this work out of the box.

```bash
make install GOBUILDIMAGE=golang:1.7.3 COMPILETAG=compile_golangimage CLARITYIMAGE=vmware/harbor-clarity-ui-builder:1.2.7
open http://registry.127.0.0.1.nip.io
```

## Run and Configure Kubernetes Auth

1. Fork and clone [kubernetes-auth](https://github.com/rcbops/kubernetes-auth).

1. Run it with the dummy backend in the same network as Harbor

    ```bash
    make run NETWORK_NAME_BASE=make_harbor
    ```

1. Work through the [kubernetes-auth example](https://github.com/rcbops/kubernetes-auth#example) to create a user.

## Modify the Rackspace Managed Kubernetes Auth code in Harbor

The auth code currently lives in the UI component. Additional targets were added to the [Makefile](Makefile) to make redeployment of the UI easy.

```bash
git checkout -b my-feature-branch rackspace-mk8s-auth

# Make your code changes.

make redeploy_ui GOBUILDIMAGE=golang:1.7.3 COMPILETAG=compile_golangimage
```

Check your code changes.

```bash
docker exec -it harbor-log tail -f /var/log/docker/$(date +%Y-%m-%d)/ui.log
```

## Release a new image

TODO

# Harbor

[![Build Status](https://travis-ci.org/vmware/harbor.svg?branch=master)](https://travis-ci.org/vmware/harbor)
[![Coverage Status](https://coveralls.io/repos/github/vmware/harbor/badge.svg?branch=master)](https://coveralls.io/github/vmware/harbor?branch=master)

**Note**: The `master` branch may be in an *unstable or even broken state* during development.
Please use [releases](https://github.com/vmware/harbor/releases) instead of the `master` branch in order to get stable binaries.

<img alt="Harbor" src="docs/img/harbor_logo.png">

Project Harbor is an enterprise-class registry server that stores and distributes Docker images. Harbor extends the open source Docker Distribution by adding the functionalities usually required by an enterprise, such as security, identity and management. As an enterprise private registry, Harbor offers better performance and security. Having a registry closer to the build and run environment improves the image transfer efficiency. Harbor supports the setup of multiple registries and has images replicated between them. In addition, Harbor offers advanced security features, such as user management, access control and activity auditing.

### Features
* **Role based access control**: Users and repositories are organized via 'projects' and a user can have different permission for images under a project.
* **Policy based image replication**: Images can be replicated (synchronized) between multiple registry instances, with auto-retry on errors. Great for load balancing, high availability, multi-datacenter, hybrid and multi-cloud scenarios.
* **Vulnerability Scanning**: Harbor scans images regularly and warns users of vulnerabilities.
* **LDAP/AD support**: Harbor integrates with existing enterprise LDAP/AD for user authentication and management.
* **Image deletion & garbage collection**: Images can be deleted and their space can be recycled.
* **Notary**: Image authenticity can be ensured.
* **Graphical user portal**: User can easily browse, search repositories and manage projects.
* **Auditing**: All the operations to the repositories are tracked.
* **RESTful API**: RESTful APIs for most administrative operations, easy to integrate with external systems.
* **Easy deployment**: Provide both an online and offline installer.

### Install & Run

**System requirements:**

**On a Linux host:** docker 1.10.0+ and docker-compose 1.6.0+ .

Download binaries of **[Harbor release ](https://github.com/vmware/harbor/releases)** and follow **[Installation & Configuration Guide](docs/installation_guide.md)** to install Harbor.

Refer to **[User Guide](docs/user_guide.md)** for more details on how to use Harbor.

### Community
**Slack:** Join Harbor's community for discussion and ask questions: [VMware {code}](https://code.vmware.com/join/), Channel: #harbor.
**Email:** harbor@ vmware.com .
More info on [partners and users](partners.md).

### Contribution
We welcome contributions from the community. If you wish to contribute code and you have not signed our contributor license agreement (CLA), our bot will update the issue when you open a pull request. For any questions about the CLA process, please refer to our [FAQ](https://cla.vmware.com/faq). Contact us for any questions: harbor @vmware.com .

### License
Harbor is available under the [Apache 2 license](LICENSE).

This project uses open source components which have additional licensing terms.  The official docker images and licensing terms for these open source components can be found at the following locations:

* Photon OS 1.0: [docker image](https://hub.docker.com/_/photon/), [license](https://github.com/vmware/photon/blob/master/COPYING)
* MySQL 5.6: [docker image](https://hub.docker.com/_/mysql/), [license](https://github.com/docker-library/mysql/blob/master/LICENSE)

### Commercial Support
If you need commercial support of Harbor, please contact us for more information: harbor@ vmware.com .



