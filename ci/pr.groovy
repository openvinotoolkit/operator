pipeline {
    agent {
      label 'ovmscheck'
    }
    stages {
        stage('Configure') {
          steps {
            script {
              checkout scm
              shortCommit = sh(returnStdout: true, script: "git log -n 1 --pretty=format:'%h'").trim()
              echo shortCommit
            }
          }
        }

        stage('style check') {
            steps {
                sh 'make style'
            }
        }

        stage('lint check') {
            steps {
                sh 'make lint'
            }
        }

        stage('docker build check') {
            steps {
                sh 'make docker_build'
            }
        }

    }
}
