FROM codercom/ubuntu-dev-go:latest
SHELL ["/bin/bash", "-c"]
RUN sudo apt-get update && \
   sudo apt-get install -y htop
LABEL project_root "~/projects/lectio"
RUN installext peterjausovec.vscode-docker
RUN installext vscode-icons-team.vscode-icons
