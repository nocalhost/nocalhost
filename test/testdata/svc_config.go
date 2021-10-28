package testdata

const SingleSvcConfig = `
name: details
serviceType: deployment
containers:
- name: ""
  dev:
    gitUrl: https://e.coding.net/nocalhost/nocalhost/bookinfo-details.git
    image: singleSvcConfig
    shell: bash
    workDir: /home/nocalhost-dev
    storageClass: ""
    resources: null
    persistentVolumeDirs: []
    command: null
    debug: null
    useDevContainer: false
    sync:
      type: send
      filePattern:
      - ./
      ignoreFilePattern:
      - .git
      - .github
    env:
    - name: DEBUG
      value: "true"
    envFrom: null
    portForward: []
`

const MultipleSvcConfig = `
- name: details
  serviceType: deployment
  containers:
  - name: ""
    dev:
      gitUrl: https://e.coding.net/nocalhost/nocalhost/bookinfo-details.git
      image: multipleSvcConfig1
      shell: bash
      workDir: /home/nocalhost-dev
      storageClass: ""
      resources: null
      persistentVolumeDirs: []
      command: null
      debug: null
      useDevContainer: false
      sync:
        type: send
        filePattern:
        - ./
        ignoreFilePattern:
        - .git
        - .github
      env:
      - name: DEBUG
        value: "true"
      envFrom: null
      portForward: []

- name: ratings
  serviceType: deployment
  containers:
  - name: ""
    dev:
      gitUrl: https://e.coding.net/nocalhost/nocalhost/bookinfo-details.git
      image: multipleSvcConfig2
      shell: bash
      workDir: /home/nocalhost-dev
      storageClass: ""
      resources: null
      persistentVolumeDirs: []
      command: null
      debug: null
      useDevContainer: false
      sync:
        type: send
        filePattern:
        - ./
        ignoreFilePattern:
        - .git
        - .github
      env:
      - name: DEBUG
        value: "true"
      envFrom: null
      portForward: []
`

const FullConfig = `
configProperties:
  version: v2

application:
  name: nocalhost
  manifestType: helmGit
  resourcePath: [ "deployments/chart" ]
  helmValues:
    - key: service.type
      value: ClusterIP

  services:
    - name: details
      serviceType: deployment
      dependLabelSelector:
        pods:
          - "app.kubernetes.io/name=mariadb"
      portForward:
        - 8080:8080
      containers:
        - name: ""
          dev:
            gitUrl: https://github.com/nocalhost/nocalhost.git
            image: fullConfig1
            workDir: /home/nocalhost-dev
            shell: "/bin/zsh"
            sync:
              type: send
              filePattern:
                - .
              ignoreFilePattern:
                - "./build"
            command:
              build: [ "make", "api" ]
              run: [ "./build/nocalhost-api", "-c", "/app/config/config.yaml" ]
              debug: [ "dlv", "--listen=:2345", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "./build/nocalhost-api", "--", "-c", "/app/config/config.yaml" ]
            debug:
              remoteDebugPort: 2345
            env:
              - name: NOCALHOST_INJECT_DEV_ENV
                value: WHATEVER
            portForward:
              - 8080:8080


    - name: ratings
      serviceType: deployment
      dependLabelSelector:
        pods:
          - "app=nocalhost-api"
      containers:
        - name: ""
          install:
            portForward:
              - 8000:80
          dev:
            gitUrl: https://e.coding.net/nocalhost/nocalhost/nocalhost-web.git
            image: fullConfig2
            workDir: /home/nocalhost-dev
            shell: "bash"
            sync:
              type: send
              filePattern:
                - .
              ignoreFilePattern:
                - ".git"
                - ".github"
                - ".vscode"
                - "node_modules"
            portForward:
              - 80:80
            resources:
              limits:
                cpu: "0.3"
                memory: 200Mi
              requests:
                cpu: "0.1"
                memory: 100Mi
`

const SingleSvcConfigCm = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: dev.nocalhost.config.bookinfo
data:
  config: |
     name: details
     serviceType: deployment
     containers:
       - name: ""
         dev:
           gitUrl: https://e.coding.net/nocalhost/nocalhost/bookinfo-details.git
           image: singleSvcConfigCm
           shell: bash
           workDir: /home/nocalhost-dev`

const MultipleSvcConfigCm = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: dev.nocalhost.config.bookinfo
data:
  config: |
    - name: details
      serviceType: deployment
      containers:
      - name: ""
        dev:
          gitUrl: https://e.coding.net/nocalhost/nocalhost/bookinfo-details.git
          image: multipleSvcConfig1Cm
          shell: bash
          workDir: /home/nocalhost-dev
          storageClass: ""
          resources: null
          persistentVolumeDirs: []
          command: null
          debug: null
          useDevContainer: false
          sync:
            type: send
            filePattern:
            - ./
            ignoreFilePattern:
            - .git
            - .github
          env:
          - name: DEBUG
            value: "true"
          envFrom: null
          portForward: []
    
    - name: ratings
      serviceType: deployment
      containers:
      - name: ""
        dev:
          gitUrl: https://e.coding.net/nocalhost/nocalhost/bookinfo-details.git
          image: multipleSvcConfig2Cm
          shell: bash
          workDir: /home/nocalhost-dev
          storageClass: ""
          resources: null
          persistentVolumeDirs: []
          command: null
          debug: null
          useDevContainer: false
          sync:
            type: send
            filePattern:
            - ./
            ignoreFilePattern:
            - .git
            - .github
          env:
          - name: DEBUG
            value: "true"
          envFrom: null
          portForward: []
`

const FullConfigCm = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: dev.nocalhost.config.bookinfo
data:
  config: |

    configProperties:
      version: v2
    
    application:
      name: nocalhost
      manifestType: helmGit
      resourcePath: [ "deployments/chart" ]
      helmValues:
        - key: service.type
          value: ClusterIP
    
      services:
        - name: details
          serviceType: deployment
          dependLabelSelector:
            pods:
              - "app.kubernetes.io/name=mariadb"
          portForward:
            - 8080:8080
          containers:
            - name: ""
              dev:
                gitUrl: https://github.com/nocalhost/nocalhost.git
                image: fullConfig1Cm
                workDir: /home/nocalhost-dev
                shell: "/bin/zsh"
                sync:
                  type: send
                  filePattern:
                    - .
                  ignoreFilePattern:
                    - "./build"
                command:
                  build: [ "make", "api" ]
                  run: [ "./build/nocalhost-api", "-c", "/app/config/config.yaml" ]
                  debug: [ "dlv", "--listen=:2345", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "./build/nocalhost-api", "--", "-c", "/app/config/config.yaml" ]
                debug:
                  remoteDebugPort: 2345
                env:
                  - name: NOCALHOST_INJECT_DEV_ENV
                    value: WHATEVER
                portForward:
                  - 8080:8080
    
    
        - name: ratings
          serviceType: deployment
          dependLabelSelector:
            pods:
              - "app=nocalhost-api"
          containers:
            - name: ""
              install:
                portForward:
                  - 8000:80
              dev:
                gitUrl: https://e.coding.net/nocalhost/nocalhost/nocalhost-web.git
                image: fullConfig2Cm
                workDir: /home/nocalhost-dev
                shell: "bash"
                sync:
                  type: send
                  filePattern:
                    - .
                  ignoreFilePattern:
                    - ".git"
                    - ".github"
                    - ".vscode"
                    - "node_modules"
                portForward:
                  - 80:80
                resources:
                  limits:
                    cpu: "0.3"
                    memory: 200Mi
                  requests:
                    cpu: "0.1"
                    memory: 100Mi
`
