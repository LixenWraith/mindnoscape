import os

def count_go_lines(directory='.'):
    total_lines = 0
    go_files_count = 0
    
    for root, _, files in os.walk(directory):
        for file in files:
            if file.endswith('.go'):
                go_files_count += 1
                file_path = os.path.join(root, file)
                with open(file_path, 'r', encoding='utf-8') as f:
                    lines = f.readlines()
                    total_lines += len(lines)
    
    return go_files_count, total_lines

if __name__ == "__main__":
    go_files_count, total_lines = count_go_lines()
    print(f"Total Go files: {go_files_count}")
    print(f"Total lines of Go code: {total_lines}")
