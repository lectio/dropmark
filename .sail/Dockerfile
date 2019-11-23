FROM codercom/ubuntu-dev-go:latest
SHELL ["/bin/bash", "-c"]
RUN sudo apt-get update && \
   sudo apt-get install -y htop
   
LABEL project_root "~/projects/lectio"

RUN installext peterjausovec.vscode-docker
RUN installext vscode-icons-team.vscode-icons
RUN installext coenraads.bracket-pair-colorizer
RUN installext esbenp.prettier-vscode
RUN installext gruntfuggly.todo-tree
RUN installext eamodio.gitlens
RUN installext yzhang.markdown-all-in-one
