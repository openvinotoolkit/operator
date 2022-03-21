#
# Copyright (c) 2022 Intel Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

import os
import sys
import re

COPYRIGHT = re.compile(r'(C|c)opyright (\((c|C)\) )?(20(0|1|2)([0-9])-)?(20(0|1|2)([0-9]),)*20(0|1|2)([0-9])(,)? ([a-z]|[A-Z]|[0-9]| )*')

def check_header(fd):
    detected = False
    try:
        for line in fd:
            if COPYRIGHT.findall(line):
                detected = True
                break
    except:
        print("ERROR: Cannot parse file:" + str(fd))
    return detected

def check_dir(start_dir):
    no_header = []

    exclude_files = ['.yaml', '__pycache__', '.vscode', '.venv', '.groovy', '.git', 'LICENSE', 'COPYING', '.md', '.png', 'NOTES.txt', '.mod', '.sum', '.tgz',
                     '.tpl', '.helmignore', 'missing_headers.txt', 'coverage.out']

    exclude_directories = ['build']

    for (d_path, _, file_set) in os.walk(start_dir):
        for f_name in file_set:
            
            skip = False
            for excluded in exclude_directories:
                if excluded in d_path:
                    skip = True
                    print('Warning - Skipping directory - ' + d_path + ' for file - ' + f_name)
                    break
                
            if skip:
                continue

            fpath = os.path.join(d_path, f_name)

            if not [test for test in exclude_files if test in fpath]:
                with open(fpath, 'r') as fd:
                    header_detected = check_header(fd)
                    if not header_detected:
                        no_header.append(fpath)
    return no_header

def main():
    if len(sys.argv) < 2:
        print('Provide start dir!')
    else:
        start_dir = sys.argv[1]
        print('Provided start dir:' + start_dir)

    print("Check for missing headers")
    no_header_set = check_dir(start_dir)

    if len(no_header_set) == 0:
        print('Success: All files have headers')
    else:
        print('#########################')
        print('## No header files detected:')
        for no_header in no_header_set:
            print(f'{no_header}')

if __name__ == '__main__':
    main()
