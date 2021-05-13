def all_images():
    cmds = {
        "piped": "piped",
        "pipecd": "pipecd",
        "pipectl": "pipectl",
        "helloworld": "helloworld",
    }
    images = {}

    for cmd, repo in cmds.items():
        images["$(DOCKER_REGISTRY)/%s:{STABLE_VERSION}" % repo] = "//cmd/%s:%s_app_image" % (cmd, cmd)

    return images
