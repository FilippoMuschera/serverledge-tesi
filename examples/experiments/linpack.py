import time
import random


MATRIX_SIZE = 300

def gaussian_elimination(A, B):

    n = len(A)
    M = [A[i][:] + [B[i]] for i in range(n)]

    for i in range(n):
        pivot = M[i][i]
        if pivot == 0: pivot = 1.0e-10

        for j in range(i + 1, n):
            factor = M[j][i] / pivot
            for k in range(i, n + 1):
                M[j][k] -= factor * M[i][k]

    x = [0.0] * n
    for i in range(n - 1, -1, -1):
        sum_ax = sum(M[i][j] * x[j] for j in range(i + 1, n))
        x[i] = (M[i][n] - sum_ax) / M[i][i]

    return x

def handler(params, context):
    n = MATRIX_SIZE

    ops = (2.0 * n * n * n) / 3.0 + (2.0 * n * n)

    A = []
    for _ in range(n):
        row = [(random.random() - 0.5) for _ in range(n)]
        A.append(row)

    B = []
    for row in A:
        B.append(sum(row))

    start = time.time()

    x = gaussian_elimination(A, B)

    duration = time.time() - start

    if duration <= 0:
        duration = 0.000001

    mflops = (ops * 1e-6 / duration)

    validity_check = abs(x[0] - 1.0) < 1e-4

    return {
        "function": "linpack_pure",
        "matrix_size": n,
        "mflops": mflops,
        "latency_seconds": duration,
        "valid": validity_check
    }