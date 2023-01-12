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

        stage('build check') {
            steps {
                sh 'make build'
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

        stage('unit check') {
            steps {
                sh 'make test-unit'
            }
        }

        stage('sdl check') {
            steps {
                sh 'make sdl-check'
            }
        }

        stage('docker build check') {
            steps {
                sh 'make build_all_images'
            }
        }

        stage("Run on commit tests") {
          steps {
              sh """
              env
              """
              build job: "ovms-operator/utils-common/ovms-o-test-on-commit", parameters: [[$class: 'StringParameterValue', name: 'OVMSOCOMMIT', value: "${GIT_COMMIT}"]]
          }    
        }
    }
}
