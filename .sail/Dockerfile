FROM codercom/ubuntu-dev-go:latest
SHELL ["/bin/bash", "-c"]
RUN sudo apt-get update && \
   sudo apt-get install -y htop
RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.34.0/install.sh | bash && \
   . ~/.nvm/nvm.sh \
   && nvm install node
RUN installext peterjausovec.vscode-docker
